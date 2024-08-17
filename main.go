package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
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

var ctx = context.Background()

func main() {
	fmt.Println("Welcome to EthScrapper for Sepolia")

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("[ERROR]		loading .env file\n")
		log.Fatalf("[ERROR]		Please properly setup .env in the root of the project (see .env.example)\n")
		return
	}

	// Get all the data from the .env file (all are #strings)
	infuraProjectId := os.Getenv("INFURA_PROJECT_ID")
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	contractAddress := os.Getenv("CONTRACT_ADDRESS")
	topic := os.Getenv("TOPIC")

	// Connect to Sepolia
	client, err := ethclient.Dial("https://sepolia.infura.io/v3/" + infuraProjectId)
	if err != nil {
		log.Fatalf("[ERROR]		Failed to connect to Ethereum node\n")
		log.Fatalf("[ERROR]		Check you Project ID (i.e. RPC-URL)\n")
		fmt.Println(err)
		return
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

	// Connect with Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: redisPassword,
		DB:       0,
	})

	// Test Redis Connection
	TestDatabase(rdb)
	retrieve(redisHost, redisPort, redisPassword)

	// Topic & latest block number:
	topicHash := common.HexToHash(topic)
	address := common.HexToAddress(contractAddress)
	blockNumberBig := big.NewInt(int64(header.Number.Uint64()))

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

	retrieve(redisHost, redisPort, redisPassword)
}

func retrieve(redisHost, redisPort, redisPassword string) {
	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: redisPassword,
		DB:       0,
	})

	// Fetch and print all keys and values
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
