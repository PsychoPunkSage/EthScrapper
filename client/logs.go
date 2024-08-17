package client

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-redis/redis/v8"
)

func QueryAndStoreLogs(client *ethclient.Client, rdb *redis.Client, contractAddress, topic string, blockNumberBig *big.Int) {
	// Topic & latest block number:
	topicHash := common.HexToHash(topic)
	address := common.HexToAddress(contractAddress)

	// query logs
	query := ethereum.FilterQuery{
		Addresses: []common.Address{address},
		Topics:    [][]common.Hash{{topicHash}},
		FromBlock: new(big.Int).Sub(blockNumberBig, big.NewInt(100)), // Only 100 blocks considered
		ToBlock:   blockNumberBig,
	}

	logs, err := client.FilterLogs(ctx, query)
	if err != nil {
		log.Fatalf("[ERROR]		Failed to query logs\n")
		fmt.Println(err)
		return
	}
	fmt.Printf("[INFO]		Found <%d> logs\n", len(logs))
	fmt.Printf("[INFO]        	- related to Topic <%v>\n", topicHash)
	fmt.Printf("[INFO]        	- in Contract Address <%v>\n", address)

	// Store logs in Redis
	index := 0
	for _, vlog := range logs {
		// fmt.Printf("Log: %+v\n", vlog)
		block, err := client.BlockByHash(ctx, vlog.BlockHash)
		if err != nil {
			log.Fatalf("[ERROR]		Failed to retrieve block: %v", err)
		}

		// Extract Data in question
		data := EventData{
			L1RootInfo: string(vlog.Data),
			Blocktime:  time.Unix(int64(block.Time()), 0),
			ParentHash: block.ParentHash(),
			LogIndex:   vlog.Index,
		}

		// Serialize the data (Go Data ==> JSON)
		serializedData, err := json.Marshal(data)
		if err != nil {
			log.Fatalf("[ERROR]        Failed to serialize data: %v", err)
		}

		// Store data in RedisDB
		err = rdb.Set(ctx, strconv.Itoa(index), serializedData, 0).Err()
		if err != nil {
			log.Fatalf("[ERROR]        Failed to store data in Redis: %v", err)
		}

		index++
	}

	log.Println("|=================================|")
	log.Println("| All events stored successfully. |")
	log.Println("|=================================|")
}
