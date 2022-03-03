package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
)

type command struct {
	ID     int           `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// create function which start a TCP server
func main() {
	// create a TCP server
	server, err := net.Listen("tcp", "localhost:55443")
	if err != nil {
		log.Fatal(err)
	}

	// start the server
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handleConn(conn)
	}
}

// handle connection
// TODO response is always okay
func handleConn(conn net.Conn) {
	defer conn.Close()

	var c command
	if err := json.NewDecoder(conn).Decode(&c); err != nil {
		fmt.Printf("error decoding response: %v\n", err)
		return
	}

	fmt.Printf("received: %+v\n", c)

	// Send a response back to person contacting us.
	conn.Write([]byte(`{"id":1,"result":["ok"]}`))
}
