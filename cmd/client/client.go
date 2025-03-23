package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

func todo() {
	panic("not implemented")
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("No conn specifier")
	}
	switch os.Args[1] {
	case "tcp":
		tcpConn()
	case "udp":
		udpConn()
	case "http":
		htp()
	case "ws":
		ws()
	default:
		log.Fatal("unsupported connection type")
	}
}
func htp() {
    req, err := http.NewRequest(http.MethodGet, "http://localhost:8080", nil)
	if err != nil {
		log.Fatalf("cant do request %s", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal("err making request", err)
	}

	log.Println("status code ", res.StatusCode)

	resBody, err := io.ReadAll(res.Body)

	if err != nil {
		log.Fatal("cliient: could not read body:", err)
	}
	log.Println("Body:",string(resBody))

}
func tcpConn() {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", ":8080")

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// Connect to the address with tcp
	conn, err := net.DialTCP("tcp", nil, tcpAddr)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Send a message to the server
	_, err = conn.Write([]byte("Hello TCP Server\n"))
	fmt.Println("send...")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Read from the connection untill a new line is send
	data, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return
	}

	// Print the data read from the connection to the terminal
	fmt.Print("> ", string(data))
}
func udpConn() {
	udpAddr, err := net.ResolveUDPAddr("udp", ":8080")

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Dial to the address with UDP
	conn, err := net.DialUDP("udp", nil, udpAddr)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Send a message to the server
	_, err = conn.Write([]byte("Hello UDP Server\n"))
	fmt.Println("send...")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Read from the connection until a newline is send
	data, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return
	}

	// Print the data read from the connection to the terminal
	fmt.Print("> ", string(data))
}
func ws() {
	url := "ws://localhost:8080/ws"
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	for {
		if err = ws.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
			return
		}
		_, msg, err := ws.ReadMessage()
		if err != nil {
			log.Fatal(err)
		}
		log.Fatal(string(msg))

	}
}
