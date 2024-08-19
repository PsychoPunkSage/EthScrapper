package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync" // CHANGED: Added for concurrency
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

type EventData struct {
	L1RootInfo string      `json:"l1RootInfo"`
	Blocktime  time.Time   `json:"blocktime"`
	ParentHash common.Hash `json:"parenthash"`
	LogIndex   uint        `json:"logIndex"`
}

type TestData struct {
	MsgData string `json:"msg"`
	Data    uint   `json:"data"`
}

var (
	ctx = context.Background()
	wg  sync.WaitGroup // CHANGED: Added for goroutine synchronization
)

func main() {
	fmt.Println("Welcome to EthScrapper for Sepolia")

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("[ERROR]		loading .env file\n")
		log.Fatalf("[ERROR]		Please properly setup .env in the root of the project (see .env.example)\n")
		return
	}

	// Get all Env variables.
	infuraProjectId := os.Getenv("INFURA_PROJECT_ID")
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	contractAddress := os.Getenv("CONTRACT_ADDRESS")
	topic := os.Getenv("TOPIC")

	// Get am Ethereum Client
	client, err := ethclient.Dial("https://sepolia.infura.io/v3/" + infuraProjectId)
	if err != nil {
		log.Fatalf("[ERROR]		Failed to connect to Ethereum node\n")
		fmt.Println(err)
		return
	}

	// Test Connection
	// chainid, err := client.ChainID(ctx)
	// if err != nil {
	// 	log.Fatalf("[ERROR]		Failed to get ChainID: %v", err)
	// }
	// fmt.Printf("[INFO]		ChainID: %d\n", chainid.Int64())

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("[ERROR]		Failed to get latest block: %v", err)
	}
	fmt.Printf("[INFO]		Latest block number: %d\n", header.Number.Uint64())

	// Getting a Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: redisPassword,
		DB:       0,
	})

	// Test Redis Client Connection
	TestDatabase(rdb)
	retrieve(rdb)

	// Get correct format of Topic, ContractAddress and Current BlockNumber
	topicHash := common.HexToHash(topic)
	address := common.HexToAddress(contractAddress)
	blockNumberBig := big.NewInt(int64(header.Number.Uint64()))

	// Query: Filter out required Topic
	query := ethereum.FilterQuery{
		Addresses: []common.Address{address},
		Topics:    [][]common.Hash{{topicHash}},
		FromBlock: new(big.Int).Sub(blockNumberBig, big.NewInt(1000)),
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

func TestDatabase(rdb *redis.Client) {
	data := TestData{
		MsgData: "test data",
		Data:    42,
	}
	serializedData, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("[ERROR]        Failed to serialize data: %v", err)
	}

	err = rdb.Set(ctx, strconv.Itoa(19112929), serializedData, 0).Err()
	if err != nil {
		log.Fatalf("[ERROR]        Failed to store data in Redis: %v", err)
	}
}

// Export key-value pairs to a .log file
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
			fmt.Printf("[WARN] Rate limit exceeded, retrying in %s...\n", backoffDuration)
			time.Sleep(backoffDuration)
		} else {
			return nil, err
		}
	}

	return nil, fmt.Errorf("failed to retrieve block after %d retries: %v", maxRetries, err)
}
