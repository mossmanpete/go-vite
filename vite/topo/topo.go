package topo

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/topo/protos"
	"gopkg.in/Shopify/sarama.v1"
	"sync"
	"time"
)

const Name = "Topo"
const CmdSet = 7
const topoCmd = 1

type TopoHandler struct {
	peers  *sync.Map
	prod   sarama.AsyncProducer
	log    log15.Logger
	term   chan struct{}
	record *cuckoofilter.CuckooFilter
	p2p    *p2p.Server
	wg     sync.WaitGroup
}

func New(addrs []string) (t *TopoHandler, err error) {
	t = &TopoHandler{
		peers:  new(sync.Map),
		log:    log15.New("module", "Topo"),
		term:   make(chan struct{}),
		record: cuckoofilter.NewCuckooFilter(1000),
	}

	if len(addrs) != 0 {
		var i, j int
		for i = 0; i < len(addrs); i++ {
			if addrs[i] != "" {
				addrs[j] = addrs[i]
				j++
			}
		}
		addrs = addrs[:j]
		if len(addrs) != 0 {
			config := sarama.NewConfig()
			prod, err := sarama.NewAsyncProducer(addrs, config)

			if err != nil {
				t.log.Error(fmt.Sprintf("create topo producer error: %v", err))
				return nil, err
			}

			t.log.Info("topo producer created")
			t.prod = prod
		}
	}

	return t, nil
}

func (t *TopoHandler) Start(svr *p2p.Server) {
	t.p2p = svr

	t.wg.Add(1)
	go t.sendLoop()
}

func (t *TopoHandler) Stop() {
	select {
	case <-t.term:
	default:
		t.log.Info("topo stop")

		close(t.term)
		t.wg.Wait()

		t.log.Info("topo stopped")
	}
}

type Peer struct {
	*p2p.Peer
	rw p2p.MsgReadWriter
}

func (t *TopoHandler) Handle(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	peer := &Peer{p, rw}
	t.peers.Store(p.String(), peer)
	defer t.peers.Delete(p.String())

	for {
		select {
		case <-t.term:
			return nil

		default:
			msg, err := rw.ReadMsg()
			if err != nil {
				t.log.Error(fmt.Sprintf("read msg error: %v", err))
				return err
			}

			if msg.Cmd != topoCmd {
				t.log.Error(fmt.Sprintf("not topoMsg cmd: %d", msg.Cmd))
				return nil
			}

			if t.Receive(msg, peer); err != nil {
				t.log.Error(fmt.Sprintf("Topo handle error: %v", err))
				return err
			}
		}
	}
}

func (t *TopoHandler) sendLoop() {
	defer t.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-t.term:
			return

		case <-ticker.C:
			monitor.LogEvent("topo", "send")
			topo := t.Topology()

			data, err := topo.Serialize()
			if err != nil {
				t.log.Error(fmt.Sprintf("serialize topo error: %v", err))
			} else {
				t.peers.Range(func(key, value interface{}) bool {
					peer := value.(*Peer)
					peer.rw.WriteMsg(&p2p.Msg{
						CmdSetID: CmdSet,
						Cmd:      topoCmd,
						Id:       0,
						Size:     uint64(len(data)),
						Payload:  data,
					})
					return true
				})

				t.write("p2p_status_event", topo.Json())
			}
		}
	}
}

// the first item is self url
func (t *TopoHandler) Topology() *Topo {
	topo := &Topo{
		Pivot: t.p2p.URL(),
		Peers: make([]string, 0, 10),
		Time:  time.Now(),
	}

	t.peers.Range(func(key, value interface{}) bool {
		p := value.(*Peer)
		topo.Peers = append(topo.Peers, p.String())
		return true
	})

	return topo
}

func (t *TopoHandler) Receive(msg *p2p.Msg, sender *Peer) (err error) {
	defer msg.Discard()

	length := len(msg.Payload)

	if length < 32 {
		err = fmt.Errorf("receive invalid topoMsg from %s@%s", sender.ID(), sender.RemoteAddr())
		return
	}

	hash := msg.Payload[:32]
	if t.record.Lookup(hash) {
		err = fmt.Errorf("has received the same topoMsg: %s", hex.EncodeToString(hash))
		return
	}

	topo := new(Topo)
	err = topo.Deserialize(msg.Payload[32:])
	if err != nil {
		t.log.Error(fmt.Sprintf("deserialize topoMsg error: %v", err))
		return
	}

	monitor.LogEvent("topo", "receive")

	t.record.InsertUnique(hash)
	// broadcast to other peer
	t.peers.Range(func(key, value interface{}) bool {
		id := key.(string)
		p := value.(*Peer)
		if id != sender.String() {
			p.rw.WriteMsg(msg)
		}
		return true
	})

	if t.prod != nil {
		monitor.LogEvent("topo", "report")
		t.write("p2p_status_event", topo.Json())
		t.log.Info("report topoMsg to kafka")
	}

	return nil
}

func (t *TopoHandler) write(topic string, data []byte) {
	t.prod.Input() <- &sarama.ProducerMessage{
		Topic:     topic,
		Value:     sarama.ByteEncoder(data),
		Timestamp: time.Now(),
	}
}

func (t *TopoHandler) Protocol() *p2p.Protocol {
	return &p2p.Protocol{
		Name:   Name,
		ID:     CmdSet,
		Handle: t.Handle,
	}
}

// @section topo
type Topo struct {
	Pivot string    `json:"pivot"`
	Peers []string  `json:"peers"`
	Time  time.Time `json:"time"`
}

// add Hash(32bit) to Front, use for determine if it has been received
func (t *Topo) Serialize() ([]byte, error) {
	data, err := proto.Marshal(&protos.Topo{
		Pivot: t.Pivot,
		Peers: t.Peers,
		Time:  t.Time.Unix(),
	})

	if err != nil {
		return nil, err
	}

	hash := crypto.Hash(32, data)

	return append(hash, data...), nil
}

func (t *Topo) Deserialize(buf []byte) error {
	pb := new(protos.Topo)
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	t.Pivot = pb.Pivot
	t.Peers = pb.Peers
	t.Time = time.Unix(pb.Time, 0)

	return nil
}

func (t *Topo) Json() []byte {
	buf, _ := json.Marshal(t)
	return buf
}
