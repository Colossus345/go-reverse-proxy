package proxy

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/Colossus345/go-reverse-proxy/internal/config"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTCPProxy(t *testing.T) {
	log.Println("START TCP TEST")
	// Start a test TCP server
	server := startTestTCPServer(t)
	defer server.Close()

	// Create proxy configuration
	cfg := &config.Config{
		Servers: []config.ServerConfig{
			{
				ListenAddr:            "localhost:60000",
				Protocol:              config.TCP,
				RemoteAddrs:           []string{server.Addr().String()},
				LoadBalancingStrategy: config.RoundRobin,
			},
		},
	}

	// Start proxy
	proxy := New(cfg)
	err := proxy.Start()
	require.NoError(t, err)
	defer proxy.Shutdown()

	// Test TCP connection
	conn, err := net.Dial("tcp", proxy.servers[0].listener.Addr().String())
	require.NoError(t, err)
	defer conn.Close()

	// Send test message
	testMsg := "Hello, TCP!"
	_, err = conn.Write([]byte(testMsg))
	require.NoError(t, err)

	// Read response
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, testMsg, string(buf[:n]))

	// Test graceful shutdown
	err = proxy.Shutdown()
	require.NoError(t, err)

	// Verify that new connections are rejected
	_, err = net.Dial("tcp", proxy.servers[0].listener.Addr().String())
	assert.Error(t, err)
}

func TestUDPProxy(t *testing.T) {
	log.Println("START UDP TEST")
	// Start a test UDP server
	server := startTestUDPServer(t)
	defer server.Close()

	// Create proxy configuration
	cfg := &config.Config{
		Servers: []config.ServerConfig{
			{
				ListenAddr:            "localhost:60001",
				Protocol:              config.UDP,
				RemoteAddrs:           []string{server.LocalAddr().String()},
				LoadBalancingStrategy: config.RoundRobin,
			},
		},
	}

	// Start proxy
	proxy := New(cfg)
	err := proxy.Start()
	require.NoError(t, err)
	defer proxy.Shutdown()

	// Test UDP connection
	udpAddr, err := net.ResolveUDPAddr("udp", cfg.Servers[0].ListenAddr)
	conn, err := net.DialUDP("udp", nil, udpAddr)
	require.NoError(t, err)
	defer conn.Close()

	// Send test message
	testMsg := "Hello, UDP!"
	_, err = conn.Write([]byte(testMsg))
	require.NoError(t, err)

	// Read response
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, testMsg, string(buf[:n]))

	// Test graceful shutdown
	err = proxy.Shutdown()
	require.NoError(t, err)
	log.Println(err)

}

func TestHTTPProxy(t *testing.T) {
	// Start a test HTTP server
	server := startTestHTTPServer(t)
	defer server.Close()

	// Create proxy configuration
	cfg := &config.Config{
		Servers: []config.ServerConfig{
			{
				ListenAddr:            "localhost:60002",
				Protocol:              config.HTTP,
				RemoteAddrs:           []string{server.Addr},
				LoadBalancingStrategy: config.RoundRobin,
			},
		},
	}

	// Start proxy
	proxy := New(cfg)
	err := proxy.Start()
	require.NoError(t, err)
	defer proxy.Shutdown()

	// Test HTTP request
	resp, err := http.Get(fmt.Sprintf("http://%s", cfg.Servers[0].ListenAddr))
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "Hello, HTTP!", string(body))
}

func TestWebSocketProxy(t *testing.T) {
	// Start a test WebSocket server
	server := startTestWSServer(t)
	defer server.Close()

	// Create proxy configuration
	cfg := &config.Config{
		Servers: []config.ServerConfig{
			{
				ListenAddr:            "localhost:60003",
				Protocol:              config.WS,
				RemoteAddrs:           []string{server.Addr},
				LoadBalancingStrategy: config.RoundRobin,
			},
		},
	}

	// Start proxy
	proxy := New(cfg)
	err := proxy.Start()
	require.NoError(t, err)
	defer proxy.Shutdown()

	// Test WebSocket connection
	wsURL := fmt.Sprintf("ws://%s/ws", cfg.Servers[0].ListenAddr)
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Send test message
	testMsg := "Hello, WebSocket!"
	err = ws.WriteMessage(websocket.TextMessage, []byte(testMsg))
	require.NoError(t, err)

	// Read response
	_, msg, err := ws.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, testMsg, string(msg))
}

func TestGracefulShutdown(t *testing.T) {
	// Start a test TCP server
	server := startTestTCPServer(t)
	defer server.Close()

	// Create proxy configuration
	cfg := &config.Config{
		Servers: []config.ServerConfig{
			{
				ListenAddr:            "localhost:60004",
				Protocol:              config.TCP,
				RemoteAddrs:           []string{server.Addr().String()},
				LoadBalancingStrategy: config.RoundRobin,
			},
		},
	}

	// Start proxy
	proxy := New(cfg)
	err := proxy.Start()
	require.NoError(t, err)
	defer proxy.Shutdown()

	// Create multiple connections
	connections := make([]net.Conn, 5)
	for i := range connections {
		conn, err := net.Dial("tcp", proxy.servers[0].listener.Addr().String())
		require.NoError(t, err)
		connections[i] = conn
	}

	// Start shutdown in a goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		proxy.Shutdown()
	}()

	// Verify that existing connections complete their work
	for _, conn := range connections {
		testMsg := "Hello, TCP!"
		_, err := conn.Write([]byte(testMsg))
		require.NoError(t, err)

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, testMsg, string(buf[:n]))
		conn.Close()
	}
}

// Helper functions to start test servers

func startTestTCPServer(t *testing.T) net.Listener {

	server, err := net.Listen("tcp", "localhost:60005")
	require.NoError(t, err)

	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				n, err := c.Read(buf)
				if err != nil {
					return
				}
				c.Write(buf[:n])
			}(conn)
		}
	}()

	return server
}

func startTestUDPServer(t *testing.T) *net.UDPConn {
	addr, err := net.ResolveUDPAddr("udp", "localhost:60006")
	require.NoError(t, err)

	server, err := net.ListenUDP("udp", addr)
	require.NoError(t, err)

	go func() {
		buf := make([]byte, 1024)
		for {
			n, clientAddr, err := server.ReadFromUDP(buf)
			if err != nil {
				return
			}
			server.WriteToUDP(buf[:n], clientAddr)
		}
	}()

	return server
}

func startTestHTTPServer(t *testing.T) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, HTTP!")
	})

	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: mux,
	}

	go server.ListenAndServe()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	return server
}

func startTestWSServer(t *testing.T) *http.Server {
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()

		for {
			_, msg, err := ws.ReadMessage()
			log.Println("Received message WS:", string(msg))
			if err != nil {
				return
			}
			ws.WriteMessage(websocket.TextMessage, msg)
		}
	})

	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: mux,
	}

	go server.ListenAndServe()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	return server
}
