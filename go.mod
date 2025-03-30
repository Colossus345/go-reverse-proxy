module github.com/Colossus345/go-reverse-proxy

go 1.24.0

toolchain go1.24.2

require (
	github.com/Colossus345/Go-interpreter v0.0.0-00010101000000-000000000000
	github.com/gorilla/websocket v1.5.3
	github.com/stretchr/testify v1.8.4
	google.golang.org/grpc v1.72.0
	google.golang.org/protobuf v1.36.5
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
)

replace github.com/Colossus345/Go-interpreter => ../Go-interpreter/
