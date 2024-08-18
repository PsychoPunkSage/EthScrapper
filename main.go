package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
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

	infuraProjectId := os.Getenv("INFURA_PROJECT_ID")
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	contractAddress := os.Getenv("CONTRACT_ADDRESS")
	topic := os.Getenv("TOPIC")

	client, err := ethclient.Dial("https://sepolia.infura.io/v3/" + infuraProjectId)
	if err != nil {
		log.Fatalf("[ERROR]		Failed to connect to Ethereum node\n")
		fmt.Println(err)
		return
	}

	chainid, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("[ERROR]		Failed to get ChainID: %v", err)
	}
	fmt.Printf("[INFO]		ChainID: %d\n", chainid.Int64())

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("[ERROR]		Failed to get latest block: %v", err)
	}
	fmt.Printf("[INFO]		Latest block number: %d\n", header.Number.Uint64())

	// CHANGED: Redis client is created once and reused
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: redisPassword,
		DB:       0,
	})

	TestDatabase(rdb)
	retrieve(rdb)

	topicHash := common.HexToHash(topic)
	address := common.HexToAddress(contractAddress)
	blockNumberBig := big.NewInt(int64(header.Number.Uint64()))

	query := ethereum.FilterQuery{
		Addresses: []common.Address{address},
		Topics:    [][]common.Hash{{topicHash}},
		FromBlock: new(big.Int).Sub(blockNumberBig, big.NewInt(100)),
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

	// CHANGED: Define batch size for Redis pipeline
	batchSize := 100
	pipe := rdb.Pipeline() // CHANGED: Start a Redis pipeline
	defer pipe.Close()

	index := 0
	for _, vlog := range logs {
		wg.Add(1)
		go func(vlog types.Log, index int) { // CHANGED: Added goroutines for concurrent processing
			defer wg.Done()

			block, err := client.BlockByHash(ctx, vlog.BlockHash)
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

			if (index+1)%batchSize == 0 || index == len(logs)-1 {
				_, err = pipe.Exec(ctx) // CHANGED: Execute the pipeline in batches
				if err != nil {
					log.Printf("[ERROR]        Failed to execute pipeline: %v", err)
				}
			}
		}(vlog, index)

		index++
	}

	wg.Wait() // CHANGED: Wait for all goroutines to finish

	log.Println("|=================================|")
	log.Println("| All events stored successfully. |")
	log.Println("|=================================|")

	retrieve(rdb)
}

func retrieve(rdb *redis.Client) {
	keys, err := rdb.Keys(ctx, "*").Result()
	if err != nil {
		log.Fatalf("Failed to fetch keys: %v", err)
	}

	log.Printf("Found <%d> keys in Redis\n", len(keys))
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
