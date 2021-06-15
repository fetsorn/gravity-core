package gravity

import (
	"encoding/json"
	"log"
	"testing"

	"github.com/Gravity-Tech/gravity-core/common/storage"
	"github.com/Gravity-Tech/gravity-core/ledger/query"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
)

func TestClient_do(t *testing.T) {
	rq := query.ByNebulaRq{
		ChainType:     6,
		NebulaAddress: "4VL4hsSPPNdqP5ajXinJ3L434uycugBxYaJiJ2Zv4FPo",
	}
	var rqi interface{}
	rqi = rq

	//ghClient, err := New("http://localhost:26657")

	// if err != nil {
	// 	log.Print(err)
	// 	t.FailNow()
	// }
	var err error
	var b []byte
	b, ok := rqi.([]byte)
	if !ok {
		b, err = json.Marshal(rq)
		if err != nil {
			log.Print(err)
			t.FailNow()
		}
	}
	client, err := rpchttp.New("http://localhost:26657", "/websocket")
	if err != nil {
		log.Print(err)
		t.FailNow()
	}

	rs, err := client.ABCIQuery(string("nebulaCustomParams"), b)

	nebulaCustomParams := storage.NebulaCustomParams{}

	err = json.Unmarshal(rs.Response.Value, &nebulaCustomParams)
	if err != nil {
		log.Print(err)
		t.FailNow()
	}

	log.Print("KARAMBA")
	log.Print(nebulaCustomParams)
	t.FailNow()
}
