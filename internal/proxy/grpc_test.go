package proxy

import (
	"context"
	"io"
	"log"
	"net"
	"testing"

	"github.com/Colossus345/go-reverse-proxy/internal/config"
	"github.com/Colossus345/go-reverse-proxy/internal/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type echoServer struct {
	proto.UnimplementedEchoServer
}

func (s *echoServer) UnaryEcho(ctx context.Context, req *proto.EchoRequest) (*proto.EchoResponse, error) {
	return &proto.EchoResponse{Message: req.Message}, nil
}

func (s *echoServer) ServerStreamEcho(req *proto.EchoRequest, stream proto.Echo_ServerStreamEchoServer) error {
	for i := 0; i < 3; i++ {
		if err := stream.Send(&proto.EchoResponse{Message: req.Message}); err != nil {
			return err
		}
	}
	return nil
}

func (s *echoServer) ClientStreamEcho(stream proto.Echo_ClientStreamEchoServer) error {
	var messages []string
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&proto.EchoResponse{Message: "Received all messages"})
		}
		if err != nil {
			return err
		}
		messages = append(messages, req.Message)
	}
}

func (s *echoServer) BidirectionalStreamEcho(stream proto.Echo_BidirectionalStreamEchoServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := stream.Send(&proto.EchoResponse{Message: req.Message}); err != nil {
			return err
		}
	}
}

func startTestGRPCServer(t *testing.T) (net.Listener, *grpc.Server) {
	lis, _ := net.Listen("tcp", "localhost:60000")

	s := grpc.NewServer()
	proto.RegisterEchoServer(s, &echoServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Errorf("failed to serve: %v", err)
		}
	}()
	return lis, s
}

func TestGRPCProxy(t *testing.T) {
	// Start a test gRPC server
	log.Println("TEST GRPC SERVER START")
	listenAddr := "localhost:3000"
	lis, server := startTestGRPCServer(t)
	defer server.Stop()

	// Create proxy configuration
	cfg := &config.Config{
		Servers: []config.ServerConfig{
			{
				ListenAddr:            listenAddr,
				Protocol:              config.GRPC,
				RemoteAddrs:           []string{lis.Addr().String()},
				LoadBalancingStrategy: config.RoundRobin,
			},
		},
	}
	log.Println("PROXY CONF", lis.Addr().String())

	// Start proxy
	proxy := New(cfg)
	err := proxy.Start()
	require.NoError(t, err)
	defer proxy.Shutdown()
	log.Println("PROXY START")

	conn, err := grpc.NewClient(listenAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	log.Println("CLIENT START")

	client := proto.NewEchoClient(conn)
	ctx := context.Background()

	// Test Unary RPC
	t.Run("UnaryEcho", func(t *testing.T) {
		log.Println("TEST UNARY")
		resp, err := client.UnaryEcho(ctx, &proto.EchoRequest{Message: "Hello, gRPC!"})
		log.Println("resp",resp)
		require.NoError(t, err)
		assert.Equal(t, "Hello, gRPC!", resp.Message)
	})

	// Test Server Streaming RPC
	t.Run("ServerStreamEcho", func(t *testing.T) {
		log.Println("TEST STREAM")
		stream, err := client.ServerStreamEcho(ctx, &proto.EchoRequest{Message: "Hello, gRPC!"})
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			resp, err := stream.Recv()
			require.NoError(t, err)
			assert.Equal(t, "Hello, gRPC!", resp.Message)
		}

		_, err = stream.Recv()
		assert.Equal(t, io.EOF, err)
	})

	// Test Client Streaming RPC
	t.Run("ClientStreamEcho", func(t *testing.T) {
		stream, err := client.ClientStreamEcho(ctx)
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			err := stream.Send(&proto.EchoRequest{Message: "Hello, gRPC!"})
			require.NoError(t, err)
		}

		resp, err := stream.CloseAndRecv()
		require.NoError(t, err)
		assert.Equal(t, "Received all messages", resp.Message)
	})

	// Test Bidirectional Streaming RPC
	t.Run("BidirectionalStreamEcho", func(t *testing.T) {
		stream, err := client.BidirectionalStreamEcho(ctx)
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			err := stream.Send(&proto.EchoRequest{Message: "Hello, gRPC!"})
			require.NoError(t, err)

			resp, err := stream.Recv()
			require.NoError(t, err)
			assert.Equal(t, "Hello, gRPC!", resp.Message)
		}

		err = stream.CloseSend()
		require.NoError(t, err)

		_, err = stream.Recv()
		assert.Equal(t, io.EOF, err)
	})
}
