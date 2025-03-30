all:
	go run cmd/server/server.go udp & echo "$$!" >server.pid

udp-server:
	go run cmd/server/server.go udp & echo "$$!" > server.pid
	go run cmd/main.go udp & echo "$$!" > proxy.pid


tcp-server:
	go run cmd/server/server.go tcp &  echo "$$!"> server.pid
	go run cmd/main.go tcp & echo "$$!" proxy.pid

ws-server:
	go run cmd/server/server.go ws & echo "$$!" server.pid
	go run cmd/main.go ws & echo "$$!" proxy.pid

http-server:
	go run cmd/server/server.go http & echo "$$!" server.pid
	go run cmd/main.go http & echo "$$!" proxy.pid
stop:
	cat server.pid | xargs kill 
	cat proxy.pid | xargs kill  
	rm server.pid
	rm proxy.pid

.PHONY: proto
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		internal/proto/echo.proto

.PHONY: test
test:
	go test -v ./... 

.PHONY: build
build:
	go build -o bin/proxy cmd/main.go

.PHONY: run
run: build
	./bin/proxy -config configs/config.yaml
