package client

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	ctx = context.Background()
	wg  sync.WaitGroup
)

type EventData struct {
	L1RootInfo string      `json:"l1RootInfo"`
	Blocktime  time.Time   `json:"blocktime"`
	ParentHash common.Hash `json:"parenthash"`
	LogIndex   uint        `json:"logIndex"`
}

func ConnectToEthereumClient(endpoint string) (*ethclient.Client, *big.Int) {
	client, err := ethclient.Dial(endpoint)
	if err != nil {
		log.Fatalf("[ERROR]		Failed to connect to Ethereum node\n")
		log.Fatalf("[ERROR]		Check you Project ID (i.e. RPC-URL)\n")
		fmt.Println(err)
		return nil, nil
	}

	chainid, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatalf("[ERROR]		Failed to get ChainID: %v", err)
	}
	fmt.Printf("[INFO]		ChainID: %d\n", chainid.Int64())

	// Check RPC connection (Print latest Block)
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("[ERROR]		Failed to get latest block: %v", err)
	}
	fmt.Printf("[INFO]		Latest block number: %d\n", header.Number.Uint64())

	blockNumberBig := big.NewInt(int64(header.Number.Uint64()))

	return client, blockNumberBig
}
