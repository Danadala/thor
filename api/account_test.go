package api_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/vechain/thor/api"
	"github.com/vechain/thor/api/utils/types"
	"github.com/vechain/thor/block"
	"github.com/vechain/thor/chain"
	"github.com/vechain/thor/genesis"
	"github.com/vechain/thor/lvldb"
	"github.com/vechain/thor/state"
	"github.com/vechain/thor/thor"
)

const (
	emptyRootHash = "56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
	testAddress   = "56e81f171bcc55a6ff8345e692c0f86e5b48e01a"
)

type account struct {
	addr    thor.Address
	balance *big.Int
	code    []byte
	storage thor.Hash
}

var accounts = []struct {
	in, want account
}{
	{
		account{thor.BytesToAddress([]byte("acc1")), big.NewInt(10), []byte{0x11, 0x12}, thor.BytesToHash([]byte("v1"))},
		account{thor.BytesToAddress([]byte("acc1")), big.NewInt(10), []byte{0x11, 0x12}, thor.BytesToHash([]byte("v1"))},
	},
	{
		account{thor.BytesToAddress([]byte("acc2")), big.NewInt(100), []byte{0x14, 0x15}, thor.BytesToHash([]byte("v2"))},
		account{thor.BytesToAddress([]byte("acc2")), big.NewInt(100), []byte{0x14, 0x15}, thor.BytesToHash([]byte("v2"))},
	},
	{
		account{thor.BytesToAddress([]byte("acc3")), big.NewInt(1000), []byte{0x20, 0x21}, thor.BytesToHash([]byte("v2"))},
		account{thor.BytesToAddress([]byte("acc3")), big.NewInt(1000), []byte{0x20, 0x21}, thor.BytesToHash([]byte("v2"))},
	},
}
var storageKey = thor.BytesToHash([]byte("key"))

func TestAccount(t *testing.T) {
	chain, db := addBestBlock(t)
	stateC := state.NewCreator(db)
	ai := api.NewAccountInterface(chain, stateC)
	router := mux.NewRouter()
	api.NewAccountHTTPRouter(router, ai)
	ts := httptest.NewServer(router)
	defer ts.Close()

	for _, v := range accounts {
		address := v.in.addr
		fmt.Println(address.String(), len(address.String()))
		res, err := http.Get(ts.URL + fmt.Sprintf("/accounts/%v", address.String()))
		if err != nil {
			t.Fatal(err)
		}
		r, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		var a types.Account
		if err := json.Unmarshal(r, &a); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, v.want.balance, a.Balance, "balance should be equal")
		assert.Equal(t, v.want.code, a.Code, "code should be equal")

		res, err = http.Get(ts.URL + fmt.Sprintf("/accounts/%v/storage?key=%v", address.String(), storageKey.String()))
		if err != nil {
			t.Fatal(err)
		}
		r, err = ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}

		value := make(map[string]string)
		if err := json.Unmarshal(r, &value); err != nil {
			t.Fatal(err)
		}
		h, err := thor.ParseHash(value[storageKey.String()])
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, v.want.storage, h, "storage should be equal")

	}

}

func addBestBlock(t *testing.T) (*chain.Chain, *lvldb.LevelDB) {
	db, _ := lvldb.NewMem()
	hash, _ := thor.ParseHash(emptyRootHash)
	s, _ := state.New(hash, db)

	for _, v := range accounts {
		address := v.in.addr
		s.SetBalance(address, v.in.balance)
		s.SetCode(address, v.in.code)
		s.SetStorage(address, storageKey, v.in.storage)
	}
	stateRoot, _ := s.Stage().Commit()
	stateC := state.NewCreator(db)
	chain := chain.New(db)
	b, err := genesis.Dev.Build(stateC)
	if err != nil {
		t.Fatal(err)
	}
	chain.WriteGenesis(b)
	best, _ := chain.GetBestBlock()
	bl := new(block.Builder).
		ParentID(best.Header().ID()).
		StateRoot(stateRoot).
		Build()
	if err := chain.AddBlock(bl, true); err != nil {
		t.Fatal(err)
	}

	return chain, db
}
