package main

import (
	"context"
	"ethscrapper/client"
	"ethscrapper/database"
	"ethscrapper/utils"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

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
	ethclient := client.ConnectToEthereumClient(fastestEndpoint)
	if ethclient == nil {
		return
	}

	// Connect with Redis
	rdbclient := database.ConnectToRedis(redisHost, redisPort, redisPassword)

	// Test Redis connection
	database.TestDatabase(rdbclient)
	database.RetrieveRedisData(redisHost, redisPort, redisPassword)

	// Query and store logs
	client.QueryAndStoreLogs(ethclient, rdbclient, contractAddress, topic)
}
