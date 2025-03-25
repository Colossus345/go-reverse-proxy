package connectionhandler

import (
	"fmt"
	"go-reverse-proxy/internal/config"
	"io"
	"log"
	"net"
)

type Message struct {
	Type config.Protocol
	Msg  []byte
}

type Connection struct {
	conn   io.ReadWriteCloser
	remote io.ReadWriteCloser
}

func SetupServer(c *config.Config) {
	tcp := Server{
		remote:      c.RemoteAddrs,
		listen_addr: c.ListenAddr,
		Type:        config.UDP,
	}
	tcp.Run()
}

type Server struct {
	Type        config.Protocol
	listen_addr string
	remote      []string
}

func (t *Server) RunUDP() error {

	udpAddr, err := net.ResolveUDPAddr("udp", t.listen_addr)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Start listening for TCP connections on the given address
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println(err)
		return err
	}
	for {
		c := t.wrapUdp(conn)
		t.handleConnection(c)
	}
	return nil
}

type UdpConn struct {
	remote_addr *net.UDPAddr
	conn        *net.UDPConn
	Mutable     bool
}

// Absolute wrong but for educational purpose ok
func (u *UdpConn) Read(b []byte) (int, error) {
	n, addr, err := u.conn.ReadFromUDP(b)
	if u.Mutable {
		u.remote_addr = addr
	}
	return n, err
}

func (u *UdpConn) Write(b []byte) (int, error) {
	if u.remote_addr != nil {
		return u.conn.WriteTo(b, u.remote_addr)
	}
	return u.conn.Write(b)
}
func (u *UdpConn) Close() error {
	return u.Close()
}

func (t *Server) wrapUdp(conn *net.UDPConn) Connection {
	udpAddr, err := net.ResolveUDPAddr("udp", t.remote[0])

	if err != nil {
		log.Fatal(err)
	}
	// Dial to the address with UDP
	remote, err := net.DialUDP("udp", nil, udpAddr)
	return Connection{conn: &UdpConn{conn: conn, Mutable: true}, remote: &UdpConn{Mutable: false, conn: remote, remote_addr: nil}}
	// return Connection{conn: conn, remote: remote}

}
func (t *Server) RunTCP() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", t.listen_addr)
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
			continue
		}
		remote, err := t.RemoteConn()
		if err != nil {
			fmt.Println(err)
			conn.Close()
			continue
		}
		// Handle new connections in a Goroutine for concurrency
		go t.handleConnection(Connection{remote: remote, conn: conn})
	}
	return nil
}

func (t *Server) Run() error {
	switch t.Type {
	case config.UDP:
		return t.RunUDP()
	case config.TCP:
		return t.RunTCP()
	default:
		return fmt.Errorf("Unsupported")
	}

}
func (t *Server) RemoteConn() (io.ReadWriteCloser, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", t.remote[0])
	if err != nil {
		log.Println(err)
		return nil, err
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	return conn, err
}

func (t *Server) handleConnection(c Connection) {
	log.Println("HANDLE")
	conn := c.conn
	remote := c.remote
	defer conn.Close()
	in_chan := make(chan Message)
	out_chan := make(chan Message)
	done := make(chan struct{})
	defer remote.Close()
	go func() {
		buf := make([]byte, 2048)
		for {
			n, err := conn.Read(buf)
			log.Println(string(buf[:n]))
			if err != nil {
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
			log.Println(string(buf[:n]))
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
					n, err := conn.Write(msg.Msg)
					log.Println("conn write", n, err)
				}
			case msg, ok := <-in_chan:
				if ok {
					log.Println(string(msg.Msg), "IN")
					n, err := remote.Write(msg.Msg)
					log.Println("remote write", n, err)
				}
			case <-done:
				return

			}
		}
	}()

	<-done
	log.Println("GRACEFUL")
}
