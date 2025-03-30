package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Colossus345/go-reverse-proxy/internal/config"
	"github.com/Colossus345/go-reverse-proxy/internal/middleware"
)

type Proxy struct {
	config *config.Config
	mu     sync.Mutex
	// Add fields for graceful shutdown
	servers    []*server
	ctx        context.Context
	cancel     context.CancelFunc
	shutdownWg sync.WaitGroup
}

type server struct {
	listener                  net.Listener
	udpConn                   *net.UDPConn
	protocol                  config.Protocol
	addr                      string
	ClientToRemoteMiddlewares []middleware.Middleware
	RemoteToClientMiddlewares []middleware.Middleware
}

func New(config *config.Config) *Proxy {
	ctx, cancel := context.WithCancel(context.Background())
	return &Proxy{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (p *Proxy) Start() error {
	// Start all servers
	for _, serverConfig := range p.config.Servers {
		s, err := p.startServer(serverConfig)
		if err != nil {
			p.Shutdown()
			return fmt.Errorf("error starting server %s: %w", serverConfig.ListenAddr, err)
		}
		p.servers = append(p.servers, s)
	}

	return nil
}

// StartWithSignal starts the proxy and returns a channel that will receive shutdown signals
func (p *Proxy) StartWithSignal() (chan os.Signal, error) {
	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if err := p.Start(); err != nil {
		return nil, err
	}

	return sigChan, nil
}

func (p *Proxy) Shutdown() error {
	// Cancel context to stop accepting new connections
	p.cancel()

	// Set a timeout for graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Create a channel to signal when shutdown is complete
	done := make(chan struct{})

	// Start shutdown in a goroutine
	go func() {
		// Wait for all active connections to finish
		p.shutdownWg.Wait()
		done <- struct{}{}
		close(done)
	}()

	// Wait for either shutdown to complete or timeout
	select {
	case <-shutdownCtx.Done():
		log.Println("Shutdown timed out, forcing close")
	case <-done:
		log.Println("Graceful shutdown completed")
	}

	// Close all listeners
	for _, s := range p.servers {
		if s.listener != nil {
			s.listener.Close()
		}
		if s.udpConn != nil {
			s.udpConn.Close()
		}
	}

	return nil
}

func (p *Proxy) startServer(serverConfig config.ServerConfig) (*server, error) {
	selector := config.NewRemoteAddrSelector(serverConfig.RemoteAddrs, serverConfig.LoadBalancingStrategy)

	// Initialize middlewares
	inboundMiddlewares := make([]middleware.Middleware, 0, len(serverConfig.InboundMiddlewares))
	log.Println("INBOUND MIDDLEWARES", serverConfig.InboundMiddlewares)
	mw, _:= middleware.New(serverConfig.InboundMiddlewares)
	inboundMiddlewares = append(inboundMiddlewares, mw)

	outboundMiddlewares := make([]middleware.Middleware, 0, len(serverConfig.OutboundMiddlewares))
	mw, _= middleware.New(serverConfig.OutboundMiddlewares)
	log.Println("outbOUND MIDDLEWARES", serverConfig.OutboundMiddlewares)
	outboundMiddlewares = append(outboundMiddlewares, mw)

	s := &server{
		protocol:                  serverConfig.Protocol,
		addr:                      serverConfig.ListenAddr,
		ClientToRemoteMiddlewares: outboundMiddlewares,
		RemoteToClientMiddlewares: inboundMiddlewares,
	}
	log.Println(s, "SERVER")

	// All protocols are handled at the transport level
	switch serverConfig.Protocol {
	case config.TCP, config.HTTP, config.WS, config.GRPC:
		listener, err := net.Listen("tcp", serverConfig.ListenAddr)
		if err != nil {
			return nil, fmt.Errorf("error starting TCP server: %w", err)
		}
		s.listener = listener

		log.Printf("%s proxy listening on %s", serverConfig.Protocol, serverConfig.ListenAddr)

		p.shutdownWg.Add(1)
		go func() {
			defer p.shutdownWg.Done()
			for {
				select {
				case <-p.ctx.Done():
					return
				default:
					conn, err := listener.Accept()
					if err != nil {
						if !isClosedError(err) {
							log.Printf("Error accepting connection: %v", err)
						}
						continue
					}

					p.shutdownWg.Add(1)
					go func() {
						defer p.shutdownWg.Done()
						s.handleTCPConnection(p.ctx, conn, selector)
					}()
				}
			}
		}()

	case config.UDP:
		addr, err := net.ResolveUDPAddr("udp", serverConfig.ListenAddr)
		if err != nil {
			return nil, fmt.Errorf("error resolving UDP address: %w", err)
		}

		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			return nil, fmt.Errorf("error starting UDP server: %w", err)
		}
		s.udpConn = conn

		log.Printf("UDP proxy listening on %s", serverConfig.ListenAddr)

		p.shutdownWg.Add(1)
		go func() {
			defer p.shutdownWg.Done()
			buffer := make([]byte, 4096)
			for {
				select {
				case <-p.ctx.Done():
					return
				default:
					n, clientAddr, err := conn.ReadFromUDP(buffer)
					if err != nil {
						if !isClosedError(err) {
							log.Printf("Error reading from UDP: %v", err)
						}
						continue
					}

					p.shutdownWg.Add(1)
					go func() {
						defer p.shutdownWg.Done()
						s.handleUDPPacket(conn, clientAddr, buffer[:n], selector)
					}()
				}
			}
		}()

	default:
		return nil, fmt.Errorf("unsupported protocol: %s", serverConfig.Protocol)
	}

	return s, nil
}

func cpy(src io.Reader, dst io.Writer, middleware middleware.MiddlewareFunc) error {
	buf := make([]byte, 4096)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			data := buf[:nr]
			var err error
			if middleware != nil {
				data, err = middleware(data)
				if err != nil {
					return err
				}
			}
			nw, ew := dst.Write(data)
			if nw < 0 || len(data) < nw {
				nw = 0
				if ew == nil {
					ew = io.ErrShortWrite
				}
			}
			if ew != nil {
				return ew
			}
			if len(data) != nw {
				return io.ErrShortWrite
			}
		}
		if er != nil {
			if er != io.EOF {
				return er
			}
			break
		}
	}
	return nil
}

func (p *server) handleTCPConnection(ctx context.Context, clientConn net.Conn, selector *config.RemoteAddrSelector) {
	defer clientConn.Close()

	remoteAddr := selector.GetNext()
	remoteConn, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		log.Printf("Error connecting to remote: %v, addr (%s)", err, remoteAddr)
		return
	}
	defer remoteConn.Close()

	log.Println(p, "SERVER")
	clientToRemote := func(data []byte) ([]byte, error) {
		for _, m := range p.ClientToRemoteMiddlewares {
			var err error
			log.Println("ClientToRemoteMiddleware", string(data))
			data, err = m.Process(data, map[string]interface{}{
				"remote": remoteAddr,
				"client": clientConn.LocalAddr(),
			})
			log.Println("ClientToRemoteMiddleware", string(data))
			if err != nil {
				return nil, err
			}
		}
		return data, nil
	}

	remoteToClient := func(data []byte) ([]byte, error) {
		for _, m := range p.RemoteToClientMiddlewares {
			var err error
			data, err = m.Process(data, map[string]interface{}{
				"remote": remoteAddr,
				"client": clientConn.LocalAddr(),
			})
			if err != nil {
				return nil, err
			}
		}
		return data, nil
	}

	done := make(chan struct{})
	go func() {
		cpy(remoteConn, clientConn, remoteToClient)
		close(done)
	}()

	cpy(clientConn, remoteConn, clientToRemote)
	select {
	case <-done:
	case <-ctx.Done():
	}
}

func (p *server) handleUDPPacket(serverConn *net.UDPConn, clientAddr *net.UDPAddr, data []byte, selector *config.RemoteAddrSelector) {
	remoteAddr := selector.GetNext()
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

// Helper function to check if an error is due to a closed connection
func isClosedError(err error) bool {
	if err == nil {
		return false
	}
	return err == io.EOF || errors.Is(err, net.ErrClosed) || err == syscall.EPIPE
}
