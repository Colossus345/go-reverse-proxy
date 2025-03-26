
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
