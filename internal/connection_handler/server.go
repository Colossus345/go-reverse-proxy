package connectionhandler

import (
	"fmt"
	"go-reverse-proxy/internal/config"
	"log"
	"net"
)

type Message struct {
	Type config.Protocol
	Msg  []byte
}

type Server interface {
	Run()
	RemoteConn()
	Close()
}

func SetupServer(c *config.Config) {

	tcp := TcpServer{
		remote: c.RemoteAddrs,
		addr:   c.ListenAddr,
	}
	tcp.Run()

}

type TcpServer struct {
	addr   string
	remote []string
}

func (t *TcpServer) Run() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", t.addr)

	if err != nil {
		fmt.Println(err)
		return err

	}

	// Start listening for TCP connections on the given address
	listener, err := net.ListenTCP("tcp", tcpAddr)

	if err != nil {
		fmt.Println(err)
		return err
	}

	for {
		// Accept new connections
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
		}
		// Handle new connections in a Goroutine for concurrency
		go t.handleConnection(conn)
	}
}
func (t *TcpServer) RemoteConn() (net.Conn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", t.remote[0])
	if err != nil {
		log.Println(err)
		return nil, err
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)

	return conn, err
}
func (t *TcpServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	remote, err := t.RemoteConn()
	in_chan := make(chan Message)
	out_chan := make(chan Message)
	done := make(chan struct{})

	if err != nil {
		log.Println(err)
		return
	}
	defer remote.Close()
	go func() {
		buf := make([]byte, 2048)
		for {
			n, err := conn.Read(buf)

			if err != nil  {
				close(in_chan)
				done <- struct{}{}
				return
			}
			in_chan <- Message{Msg: buf[:n]}
		}
	}()
	go func() {
		buf := make([]byte, 2048)
		for {
			n, err := remote.Read(buf)

			if err != nil {
				close(out_chan)
				done <- struct{}{}
				return
			}
			out_chan <- Message{Msg: buf[:n]}
		}
	}()
	go func() {
		for {
			select {
			case msg, ok := <-out_chan:
				if ok {
					log.Println(string(msg.Msg), "OUT")
					conn.Write(msg.Msg)
				}
			case msg, ok := <-in_chan:
				if ok {
					log.Println(string(msg.Msg), "IN")
					remote.Write(msg.Msg)
				}
			case <-done:
				return

			}
		}

	}()

	<-done
    log.Println("GRACEFUL")
}

