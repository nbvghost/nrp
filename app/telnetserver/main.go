package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	if len(os.Args) == 3 {
		l, err := net.Listen("tcp", fmt.Sprintf("%s:%s", os.Args[1], os.Args[2]))
		if err != nil {
			panic(err)
		}
		defer l.Close()

		for {
			conn, err := l.Accept()
			if err != nil {
				panic(err)
			}

			go read(conn)
		}
	}
}
func read(conn net.Conn) {
	for {
		tempBuf := make([]byte, 4096)
		n, err := conn.Read(tempBuf)
		if err != nil {
			return
		}
		if n == 0 {
			continue
		}
		body := tempBuf[:n]
		log.Println(string(body))
		conn.Write(body)
	}
}
