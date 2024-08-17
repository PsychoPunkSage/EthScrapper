# EthScrapper

>> Major refactor of the codebase, seperated all the components into different `packages` to keep things clean in `main.go` file.

## Demo
> **Payload:**<br>
>   - `contract address` : 0x919Ab642766D1a015F546811F15d5DB324F5E415
>   - `topic hash` : 0x3e54d0825ed78523037d00a81759237eb436ce774bd546993ee67a1b67b6e766

```sh
make start
```

<details>
<summary>
Output
</summary>


```json
a3c253ee34c6bf9efae59b3ffc75da226803081ddc04e3928812307f14629f8b
Welcome to EthScrapper for Sepolia
[ERROR | utils]		pinging endpoint https://eth-sepolia.g.alchemy.com/v2/XXXXXXXXXX_XXXXXXXXXXXXXXXXXX-k1: Get "https://eth-sepolia.g.alchemy.com/v2/XXXXXXXXXX_XXXXXXXXXXXXXXXXXX-k1": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
[ERROR | utils]		pinging endpoint https://sepolia.infura.io/v3/XXXXXXXXXXXXXXXXXXXXXXXXXXX: Get "https://sepolia.infura.io/v3/XXXXXXXXXXXXXXXXXXXXXXXXXXX": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
[FASTEST] Selected endpoint: https://sepolia.infura.io/v3/XXXXXXXXXXXXXXXXXXXXXXXXXXX
[INFO]		ChainID: 11155111
[INFO]		Latest block number: 6515950
2024/08/17 11:45:39 Key: 19, Value: {"msg":"test data","data":42}
[INFO]		Found <31> logs
[INFO]        	- related to Topic <0x3e54d0825ed78523037d00a81759237eb436ce774bd546993ee67a1b67b6e766>
[INFO]        	- in Contract Address <0x919Ab642766D1a015F546811F15d5DB324F5E415>
2024/08/17 11:47:03 |=================================|
2024/08/17 11:47:03 | All events stored successfully. |
2024/08/17 11:47:03 |=================================|
```

</details>

## Key Highlights

* **client**: Contains code related to `Client side operation` and `Query and Store`
* **database**: Contains all the operation related to `Database connection & Health Check`
* **utils**: Logic to find `fastest RPC URL`

## Reqirements
1. `Go`:

<details>
<summary>
Installtion
</summary>

1. Install Go version 1.16 or above.

2. Define GOPATH environment variable and modify PATH to access your Go binaries. A common setup is as follows. You could always specify it based on your own flavor.

```sh
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
```

</details>

2. `Docker`: [Installation](https://docs.docker.com/engine/install/)
3. `API Key`: Visit `Infura` or `Alchemy`

## Installation

### 1. Setup `.env` file

> Set up a `.env` file in the root of your project directory. Follow `.env.example` to create the `.env` file.

By default, passowrd of RedisDB is `ethscrapper` (use it in `.env` file). You can change it in **`redis.conf`** (`./conf/redis.conf`) 

### 2. Build the project

> There is a `Makefile` in the project directory, you can have a look at it to get a detailed overview of all the commands.

**Build**
```sh
make build
```

**Start** (build + launch docker image + run script) - Prefered to use One time
```sh
make start
```

**Run** (build + run script) - Can use is multiple times after `make start`
```sh
make run
```

**Stop** (Kills redis image) - Once work is done, stop everything
```sh
make stop
```