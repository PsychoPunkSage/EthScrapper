build:
	@go build -o bin/fs

run: build start-redis
	@./bin/fs

test:
	@go test ./... -v

start-redis:
	@sudo docker run --name redis -d --rm \
		-v $(PWD)/conf/redis.conf:/usr/local/etc/redis/redis.conf \
		-p 6379:6379 redis redis-server /usr/local/etc/redis/redis.conf

stop-redis:
	@sudo docker stop redis