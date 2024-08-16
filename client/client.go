package client

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var ctx = context.Background()

type EventData struct {
	L1RootInfo string      `json:"l1RootInfo"`
	Blocktime  time.Time   `json:"blocktime"`
	ParentHash common.Hash `json:"parenthash"`
	LogIndex   uint        `json:"logIndex"`
}

func ConnectToEthereumClient(endpoint string) *ethclient.Client {
	client, err := ethclient.Dial(endpoint)
	if err != nil {
		log.Fatalf("[ERROR]		Failed to connect to Ethereum node\n")
		log.Fatalf("[ERROR]		Check you Project ID (i.e. RPC-URL)\n")
		fmt.Println(err)
		return nil
	}

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("[ERROR]		Failed to get latest block: %v", err)
	}
	fmt.Printf("Latest block number: %d\n", header.Number.Uint64())

	return client
}
