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

func QueryAndStoreLogs(client *ethclient.Client, rdb *redis.Client, contractAddress, topic string) {
	// Topic and Address:
	address := common.HexToAddress(contractAddress)
	topicHash := common.HexToHash(topic)

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("[ERROR]		Failed to get latest block: %v", err)
	}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{address},
		Topics:    [][]common.Hash{{topicHash}},
		FromBlock: big.NewInt(0), // Start from the first block on Sepolia
		ToBlock:   big.NewInt(int64(header.Number.Uint64())),
	}

	logs, err := client.FilterLogs(ctx, query)
	if err != nil {
		log.Fatalf("[ERROR]		Failed to query logs\n")
		fmt.Println(err)
		return
	}
	fmt.Printf("Found %d logs\n", len(logs))

	// If no logs are found, try without filtering by topics to verify logs exist
	if len(logs) == 0 {
		fmt.Println("No logs found with topic filter, querying without topics...")
		query.Topics = nil // Remove topic filter

		logs, err = client.FilterLogs(ctx, query)
		if err != nil {
			fmt.Println("[ERROR]		Failed to query logs without topics")
			fmt.Println(err)
			return
		}
		fmt.Printf("Found %d logs without topics\n", len(logs))
	}

	// Store logs in Redis
	index := 0
	for _, vlogs := range logs {
		fmt.Printf("Log: %+v\n", vlogs)
		block, err := client.BlockByHash(ctx, vlogs.BlockHash)
		if err != nil {
			log.Fatalf("[ERROR]		Failed to retrieve block: %v", err)
		}

		// Extract Data in question
		data := EventData{
			L1RootInfo: string(vlogs.Data),
			Blocktime:  time.Unix(int64(block.Time()), 0),
			ParentHash: block.ParentHash(),
			LogIndex:   vlogs.Index,
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
