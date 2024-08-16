package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-redis/redis"
	"github.com/joho/godotenv"
)

type EventData struct {
	L1RootInfo string      `json:"l1RootInfo"`
	Blocktime  time.Time   `json:"blocktime"`
	ParentHash common.Hash `json:"parenthash"`
	LogIndex   uint        `json:"logIndex"`
}

var ctx = context.Background()

func main() {
	fmt.Println("Welcome to EthScrapper for Sepolia")

	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error:: loading .env file")
		fmt.Println("Error:: Please properly setup .env in the root of the project (see .env.example) ")
		return
	}

	// Get all the data from the .env file (all are #strings)
	alchemyProjectId := os.Getenv("ALCHEMY_PROJECT_ID")
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

	// Connect to Sepolia
	client, err := ethclient.Dial(alchemyProjectId)
	if err != nil {
		fmt.Println("Error:: Failed to connect to Ethereum node")
		fmt.Println("Error:: Check you Project ID (i.e. RPC-URL)")
		fmt.Println(err)
		return
	}

	// Connect with Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: redisPassword,
		DB:       0,
	})

	// Topic and Address:
	address := common.HexToAddress(contractAddress)
	topicHash := common.HexToHash(topic)

	// query logs
	query := ethereum.FilterQuery{
		Addresses: []common.Address{address},
		Topics:    [][]common.Hash{{topicHash}},
	}
}
