package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"time"
)

func main() {

	clientUdpServer()

	/*clientAddr, clientErr := net.ResolveUDPAddr("udp", ":34599")
	if glog.Error(clientErr) {
		os.Exit(1)
	}


	clientConn, err := net.ListenUDP("udp", clientAddr)
	if glog.Error(err) {
		os.Exit(1)
	}


	go clientUdpServer()


	for {
		// Here must use make and give the lenth of buffer
		data := make([]byte,64)
		n, rAddr, err := clientConn.ReadFromUDP(data)
		if glog.Error(err) {

			continue
		}

		n, err = clientConn.WriteToUDP(data[0:n],rAddr)
		if glog.Error(err) {
			continue
		}

	}*/

}
func clientUdpServer() {

	plAddr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: 7770}
	prAddr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: 3389}

	//proxyConn, err := net.Dial("udp",  ":3389")
	proxyConn, err := net.DialUDP("udp", plAddr, prAddr)
	log.Println(err)

	go func() {

		for {
			proxyConn.Write([]byte("52645456"))
			if err != nil {

			}
			time.Sleep(1000 * time.Millisecond)
		}
	}()

	var i int32 = 1

	for {
		data := make([]byte, 64)
		n, err := proxyConn.Read(data)

		buffer := bytes.NewBuffer(make([]byte, 0))

		binary.Read(buffer, binary.BigEndian, &i)

		log.Println("读取代理数据", n, i)
		if err != nil {
			continue
		}

		i++
		buffer.Reset()
		binary.Write(buffer, binary.BigEndian, &i)

		//fmt.Print(buffer.Bytes())
		n, err = proxyConn.Write(buffer.Bytes())
		if err != nil {
			continue
		}

	}

}
