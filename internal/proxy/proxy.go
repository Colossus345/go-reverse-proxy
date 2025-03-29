package proxy

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/Colossus345/go-reverse-proxy/internal/config"
)

type Proxy struct {
	config *config.Config
}

func New(config *config.Config) *Proxy {
	return &Proxy{
		config: config,
	}
}

func (p *Proxy) Start() error {
	for _, server := range p.config.Servers {
		go func(s config.ServerConfig) {
			if err := p.startServer(s); err != nil {
				log.Printf("Error starting server %s: %v", s.ListenAddr, err)
			}
		}(server)
	}
	return nil
}

func (p *Proxy) startServer(server config.ServerConfig) error {
	switch server.Protocol {
	case config.TCP:
		return p.startTCPServer(server)
	case config.UDP:
		return p.startUDPServer(server)
	default:
		return fmt.Errorf("unsupported protocol: %s", server.Protocol)
	}
}

func (p *Proxy) startTCPServer(server config.ServerConfig) error {
	listener, err := net.Listen("tcp", server.ListenAddr)
	if err != nil {
		return fmt.Errorf("error starting TCP server: %w", err)
	}
	defer listener.Close()

	log.Printf("TCP proxy listening on %s", server.ListenAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go p.handleTCPConnection(conn, server.RemoteAddr)
	}
}

func (p *Proxy) startUDPServer(server config.ServerConfig) error {
	addr, err := net.ResolveUDPAddr("udp", server.ListenAddr)
	if err != nil {
		return fmt.Errorf("error resolving UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("error starting UDP server: %w", err)
	}
	defer conn.Close()

	log.Printf("UDP proxy listening on %s", server.ListenAddr)

	buffer := make([]byte, 4096)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading from UDP: %v", err)
			continue
		}

		go p.handleUDPPacket(conn, clientAddr, buffer[:n], server.RemoteAddr)
	}
}

func (p *Proxy) handleTCPConnection(clientConn net.Conn, remoteAddr string) {
	defer clientConn.Close()

	remoteConn, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		log.Printf("Error connecting to remote: %v", err)
		return
	}
	defer remoteConn.Close()

	// Create bidirectional copy
	go func() {
		if _, err := io.Copy(remoteConn, clientConn); err != nil {
			log.Printf("Error copying from client to remote: %v", err)
		}
	}()

	if _, err := io.Copy(clientConn, remoteConn); err != nil {
		log.Printf("Error copying from remote to client: %v", err)
	}
}

func (p *Proxy) handleUDPPacket(serverConn *net.UDPConn, clientAddr *net.UDPAddr, data []byte, remoteAddr string) {
	remoteConn, err := net.Dial("udp", remoteAddr)
	if err != nil {
		log.Printf("Error connecting to remote UDP: %v", err)
		return
	}
	defer remoteConn.Close()

	// Send data to remote
	if _, err := remoteConn.Write(data); err != nil {
		log.Printf("Error writing to remote: %v", err)
		return
	}

	// Read response from remote
	response := make([]byte, 4096)
	n, err := remoteConn.Read(response)
	if err != nil {
		log.Printf("Error reading from remote: %v", err)
		return
	}

	// Send response back to client
	if _, err := serverConn.WriteToUDP(response[:n], clientAddr); err != nil {
		log.Printf("Error writing response to client: %v", err)
	}
} 