package client

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
		FromBlock: new(big.Int).Sub(blockNumberBig, big.NewInt(1000)), // Only 100 blocks considered
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

	// Define batch size for Redis pipeline (To enable Batch writing to make things Fast.)
	batchSize := 100
	pipe := rdb.Pipeline()
	defer pipe.Close()

	// Store logs in Redis
	index := 0
	for _, vlog := range logs {
		wg.Add(1)
		go func(vlog types.Log, index int) {
			defer wg.Done()

			block, err := fetchBlockWithRetry(client, vlog.BlockHash, 5)
			if err != nil {
				log.Printf("[ERROR]		Failed to retrieve block: %v", err)
				return
			}

			data := EventData{
				L1RootInfo: string(vlog.Data),
				Blocktime:  time.Unix(int64(block.Time()), 0),
				ParentHash: block.ParentHash(),
				LogIndex:   vlog.Index,
			}

			serializedData, err := json.Marshal(data)
			if err != nil {
				log.Printf("[ERROR]        Failed to serialize data: %v", err)
				return
			}

			key := strconv.Itoa(index)
			pipe.Set(ctx, key, serializedData, 0) // CHANGED: Add to pipeline instead of setting directly

			if (index+1)%batchSize == 0 /* || index == len(logs)-1*/ {
				_, err = pipe.Exec(ctx) // Execute the pipeline in batches
				if err != nil {
					log.Printf("[ERROR]        Failed to execute pipeline: %v", err)
				}
			}
		}(vlog, index)

		index++
	}

	wg.Wait()

	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("[ERROR]        Failed to execute final pipeline: %v", err)
	}

	log.Println("|=================================|")
	log.Println("| All events stored successfully. |")
	log.Println("|=================================|")
	retrieve(rdb)
	exportDataToLogFile(rdb, "redis.log")
}

func fetchBlockWithRetry(client *ethclient.Client, blockHash common.Hash, maxRetries int) (*types.Block, error) {
	var block *types.Block
	var err error

	for i := 0; i < maxRetries; i++ {
		block, err = client.BlockByHash(ctx, blockHash)
		if err == nil {
			return block, nil
		}

		// Check if the error is a rate limit error
		if strings.Contains(err.Error(), "429 Too Many Requests") {
			backoffDuration := time.Duration(i+1) * time.Second
			fmt.Printf("[WARN]		Rate limit exceeded, retrying in %s...\n", backoffDuration)
			time.Sleep(backoffDuration)
		} else {
			return nil, err
		}
	}

	return nil, fmt.Errorf("failed to retrieve block after %d retries: %v", maxRetries, err)
}

func retrieve(rdb *redis.Client) {
	keys, err := rdb.Keys(ctx, "*").Result()
	if err != nil {
		log.Fatalf("Failed to fetch keys: %v", err)
	}

	log.Printf("Found <%d> keys in Redis\n", len(keys))

	//// Uncomment below code if you want to see key, value pairs being printed.

	// for _, key := range keys {
	// 	val, err := rdb.Get(ctx, key).Result()
	// 	if err != nil {
	// 		log.Fatalf("Failed to fetch value for key %s: %v", key, err)
	// 	}
	// 	log.Printf("Key: %s, Value: %s\n", key, val)
	// }
}

func exportDataToLogFile(rdb *redis.Client, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create log file: %v", err)
	}
	defer file.Close()

	keys, err := rdb.Keys(ctx, "*").Result()
	if err != nil {
		log.Fatalf("Failed to fetch keys: %v", err)
	}

	for _, key := range keys {
		val, err := rdb.Get(ctx, key).Result()
		if err != nil {
			log.Printf("Failed to fetch value for key %s: %v", key, err)
			continue
		}

		logEntry := fmt.Sprintf("Key: %s, Value: %s\n", key, val)
		_, err = file.WriteString(logEntry)
		if err != nil {
			log.Printf("Failed to write log entry: %v", err)
		}
	}

	log.Printf("Exported %d key-value pairs to %s\n", len(keys), filename)
}
