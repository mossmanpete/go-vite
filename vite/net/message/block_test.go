package message

import (
	crand "crypto/rand"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	mrand "math/rand"
	"testing"
	"time"
)

// GetAccountBlocks
func mockGetAccountBlocks() GetAccountBlocks {
	var ga GetAccountBlocks

	ga.From.Height = mrand.Uint64()
	crand.Read(ga.From.Hash[:])

	ga.Count = mrand.Uint64()
	ga.Forward = mrand.Intn(10) > 5

	crand.Read(ga.Address[:])

	return ga
}

func equalGetAccountBlocks(g, g2 GetAccountBlocks) bool {
	if g.From.Height != g2.From.Height {
		return false
	}

	if g.From.Hash != g2.From.Hash {
		return false
	}

	if g.Count != g2.Count {
		return false
	}

	if g.Forward != g2.Forward {
		return false
	}

	if g.Address != g2.Address {
		return false
	}

	return true
}

func TestGetAccountBlocks_Serialize(t *testing.T) {
	ga := mockGetAccountBlocks()

	buf, err := ga.Serialize()
	if err != nil {
		t.Error(err)
	}

	var g GetAccountBlocks
	err = g.Deserialize(buf)
	if err != nil {
		t.Error(err)
	}

	if !equalGetAccountBlocks(ga, g) {
		t.Fail()
	}
}

func mockGetSnapshotBlocks() GetSnapshotBlocks {
	var ga GetSnapshotBlocks

	ga.From.Height = mrand.Uint64()
	crand.Read(ga.From.Hash[:])

	ga.Count = mrand.Uint64()
	ga.Forward = mrand.Intn(10) > 5

	return ga
}

func equalGetSnapshotBlocks(g, g2 GetSnapshotBlocks) bool {
	if g.From.Height != g2.From.Height {
		return false
	}

	if g.From.Hash != g2.From.Hash {
		return false
	}

	if g.Count != g2.Count {
		return false
	}

	if g.Forward != g2.Forward {
		return false
	}

	return true
}

func TestGetSnapshotBlocks_Deserialize(t *testing.T) {
	gs := mockGetSnapshotBlocks()

	buf, err := gs.Serialize()
	if err != nil {
		t.Error(err)
	}

	var g GetSnapshotBlocks
	err = g.Deserialize(buf)
	if err != nil {
		t.Error(err)
	}

	if !equalGetSnapshotBlocks(gs, g) {
		t.Fail()
	}
}

// AccountBlocks
func mockAccountBlocks() AccountBlocks {
	var ga AccountBlocks

	n := mrand.Intn(100)
	ga.Blocks = make([]*ledger.AccountBlock, n)

	for i := 0; i < n; i++ {
		ga.Blocks[i] = &ledger.AccountBlock{
			Meta:           &ledger.AccountBlockMeta{},
			BlockType:      0,
			Hash:           types.Hash{},
			Height:         0,
			PrevHash:       types.Hash{},
			AccountAddress: types.Address{},
			PublicKey:      nil,
			ToAddress:      types.Address{},
			FromBlockHash:  types.Hash{},
			Amount:         new(big.Int),
			TokenId:        types.TokenTypeId{},
			Quota:          0,
			Fee:            new(big.Int),
			SnapshotHash:   types.Hash{},
			Data:           nil,
			Timestamp:      &time.Time{},
			StateHash:      types.Hash{},
			LogHash:        nil,
			Difficulty:     nil,
			Nonce:          nil,
			Signature:      nil,
		}
	}

	return ga
}

func equalAccountBlocks(g, g2 AccountBlocks) bool {
	if len(g.Blocks) != len(g2.Blocks) {
		return false
	}

	return true
}

func TestAccountBlocks_Deserialize(t *testing.T) {
	gs := mockAccountBlocks()

	buf, err := gs.Serialize()
	if err != nil {
		t.Error(err)
	}

	var g AccountBlocks
	err = g.Deserialize(buf)
	if err != nil {
		t.Error(err)
	}

	if !equalAccountBlocks(gs, g) {
		t.Fail()
	}
}

func TestSubLedger_Serialize(t *testing.T) {
	s := new(SubLedger)
	buf, err := s.Serialize()
	if err != nil {
		t.Error(err)
	}

	s2 := new(SubLedger)
	err = s2.Deserialize(buf)
	if err != nil {
		t.Error(err)
	}
}
