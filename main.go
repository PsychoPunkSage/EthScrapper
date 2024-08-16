package main

import (
	"context"
	"encoding/json"
	"ethscrapper/utils"
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
	rpcEnpoints := []string{
		os.Getenv("INFURA_PROJECT_ID"),
		os.Getenv("ALCHEMY_PROJECT_ID"),
	}
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	contractAddress := os.Getenv("CONTRACT_ADDRESS")
	topic := os.Getenv("TOPIC")

	// fmt.Println(reflect.TypeOf(alchemyProjectId))
	// fmt.Println(reflect.TypeOf(redisHost))
	// fmt.Println(reflect.TypeOf(redisPort))
	// fmt.Println(reflect.TypeOf(redisPassword))
	// fmt.Println(reflect.TypeOf(contractAddress))
	// fmt.Println(reflect.TypeOf(topic))

	// Get Minimum latencies
	fastestEndpoint := utils.SelectFastestRPC(rpcEnpoints, utils.Measurelatency(rpcEnpoints))
	fmt.Printf("[FASTEST] Selected endpoint: %s\n", fastestEndpoint)

	// Connect to Fastest RPC Client
	client, err := ethclient.Dial("https://sepolia.infura.io/v3/" + infuraProjectId)
	if err != nil {
		log.Fatalf("[ERROR]		Failed to connect to Ethereum node\n")
		log.Fatalf("[ERROR]		Check you Project ID (i.e. RPC-URL)\n")
		fmt.Println(err)
		return
	}

	// Check RPC connection (Print latest Block)
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("[ERROR]		Failed to get latest block: %v", err)
	}
	fmt.Printf("Latest block number: %d\n", header.Number.Uint64())

	// Connect with Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: redisPassword,
		DB:       0,
	})

	// Test Redis Connection
	TestDatabase(rdb)
	retrieve(redisHost, redisPort, redisPassword)

	// Topic and Address:
	address := common.HexToAddress(contractAddress)
	topicHash := common.HexToHash(topic)

	// query logs
	query := ethereum.FilterQuery{
		Addresses: []common.Address{address},
		Topics:    [][]common.Hash{{topicHash}},
		FromBlock: big.NewInt(0), // Start from the first block on Sepolia
		ToBlock:   big.NewInt(int64(header.Number.Uint64())),
	}
	// fmt.Printf("Found Query: \n%v \n", query)

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

	for _, key := range keys {
		val, err := rdb.Get(ctx, key).Result()
		if err != nil {
			log.Fatalf("Failed to fetch value for key %s: %v", key, err)
		}
		log.Printf("Key: %s, Value: %s\n", key, val)
	}
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

	err = rdb.Set(ctx, strconv.Itoa(19), serializedData, 0).Err()
	if err != nil {
		log.Fatalf("[ERROR]        Failed to store data in Redis: %v", err)
	}
}
