package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func home(w http.ResponseWriter, r *http.Request) {
	log.Println("RECEIVED")
	fmt.Fprintf(w, "home page")
}
func wsEndPoint(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer ws.Close()
	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			fmt.Println("read err:", err)
			break
		}
		fmt.Println("Received:", msg)
		if err := ws.WriteMessage(websocket.TextMessage, msg); err != nil {
			fmt.Println("write err:", err)
			break
		}
	}

}
func setupRoutes() {
	http.HandleFunc("/", home)
	http.HandleFunc("/ws", wsEndPoint)
}
func httpConn() {
	setupRoutes()
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func tcpConn() {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
		}
		go handleConn(conn)
	}
}
func handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		data, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}

		// Print the data read from the connection to the terminal
		fmt.Print("> ", string(data))

		// Write back the same message to the client.
		conn.Write([]byte(data))
	}
}
func udpConn() {
	udpAddr, err := net.ResolveUDPAddr("udp", ":8080")

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Start listening for UDP packages on the given address
	conn, err := net.ListenUDP("udp", udpAddr)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Read from UDP listener in endless loop
	for {
		var buf [512]byte
		_, addr, err := conn.ReadFromUDP(buf[0:])
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Print("> ", string(buf[0:]))

		conn.WriteToUDP(buf[0:], addr)
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("No type specified")
	}
	switch os.Args[1] {
	case "http", "ws":
		{
			httpConn()
		}
	case "tcp":
		{
			tcpConn()
		}
	case "udp":
		{
			udpConn()
		}
	default:
		log.Fatal("unsupported type ", os.Args[1])
	}
}
