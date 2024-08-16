# EthScrapper

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

**Run** (build + run Docker image of Redis)
```sh
make run
```

**Kill Image**
```sh
make stop-redis
```