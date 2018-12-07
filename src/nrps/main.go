package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"runtime"
	"strings"
	"sync"
	"time"
)

var outList OutList
var globalID uint64
var nrps Nrps
var clientConnect *net.TCPConn

type OutList struct {
	Data map[uint64]*HttpPack
	Lock sync.Mutex
}
func (ol *OutList) Set(key uint64,value *HttpPack) {
	ol.Lock.Lock()
	defer ol.Lock.Unlock()
	if ol.Data==nil{
		ol.Data = make(map[uint64]*HttpPack)
	}
	ol.Data[key] = value
}
func (ol *OutList) Get(key uint64) (*HttpPack,bool)  {
	ol.Lock.Lock()
	defer ol.Lock.Unlock()
	if ol.Data==nil{
		ol.Data = make(map[uint64]*HttpPack)
	}
	_,ok:=ol.Data[key]
	return ol.Data[key],ok
}
func (ol *OutList) Del(key uint64)  {
	ol.Lock.Lock()
	defer ol.Lock.Unlock()
	delete(ol.Data,key)
}
type Nrps struct {
	BindPort string
	HttpPort string
}
type HttpPack struct {
	out  chan []byte
	time time.Time
}

func init() {

	/*go func() {

		for _, value := range outList {
			//outList
			if time.Now().Unix()-value.time.Unix() > 10 {
				close(value.out)
			}
		}

	}()*/
}

func startWeb() {
	l, err := net.Listen("tcp", nrps.HttpPort)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	for {

		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}

		go readweb(conn)
	}
}
func CheckError(err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		log.Println(file, line, err)
	}
}
var lock sync.RWMutex
func readweb(conn net.Conn) {
	defer conn.Close()
	//fmt.Println(conn.LocalAddr())
	reader := bufio.NewReader(conn)

	resp, err := http.ReadRequest(reader)
	if resp == nil {
		//fmt.Println(resp)
		return
	}
	CheckError(err)
	//fmt.Println(resp.Method)
	//fmt.Println(resp.Host)
	//fmt.Println(resp)

	if strings.EqualFold(resp.Method, http.MethodConnect) {
		//fmt.Println(resp.Response)
		//fmt.Println(http.ReadResponse(reader,resp))

		server, err := net.DialTimeout("tcp", resp.Host, time.Second*12)

		if err != nil {

			return
		}

		//http.StatusOK

		fmt.Fprint(conn, "HTTP/1.1 200 Connection established\r\n\r\n")

		go func() {
			_, err := io.Copy(server, conn)
			CheckError(err)
		}()
		_, err = io.Copy(conn, server)
		CheckError(err)

	} else {
		dfs, err := httputil.DumpRequest(resp, true)
		//fmt.Println(string(dfs))
		CheckError(err)

		if clientConnect != nil {
			//clientConnect.Write(b)


			lock.Lock()
			var id = globalID + 1
			globalID=id
			lock.Unlock()

			lenght := int32(len(dfs))
			buffer := bytes.NewBuffer(make([]byte, 0))
			binary.Write(buffer, binary.LittleEndian, &id)     //8
			binary.Write(buffer, binary.LittleEndian, &lenght) //4
			binary.Write(buffer, binary.LittleEndian, &dfs)

			//fmt.Println(len(dfs))
			hp := &HttpPack{out: make(chan []byte), time: time.Now()}
			outList.Set(id,hp)

			clientConnect.Write(buffer.Bytes())


			fmt.Println("数据写给客户端，等待客户端返回")

			h,ok:=outList.Get(id)
			if ok{
				select {
				case bdfd := <-h.out:
					conn.Write(bdfd)
					//close(hp.out)
					fmt.Println("-----------输出数据-----------------")
				case <-time.After(10*time.Second):
					conn.Write([]byte("timeout"))
					//close(hp.out)
					fmt.Println("-----------客户端返回超时-----------------")

				}
			}else{
				conn.Write([]byte("数据出错"))
				//close(hp.out)
				fmt.Println("-----------数据出错-----------------")
			}
			outList.Del(id)
			//fmt.Println(string(bdfd))

		} else {

			resp, err := http.DefaultTransport.RoundTrip(resp)
			if err != nil {

				return
			}
			defer resp.Body.Close()

			b, err := httputil.DumpResponse(resp, true)

			conn.Write(b)

		}
	}

}

func main() {

	b, err := ioutil.ReadFile("nrps.json")
	if err != nil {
		panic(err)
	}

	json.Unmarshal(b, &nrps)

	//fmt.Println(nrps)

	go startWeb()

	tcpAddress, err := net.ResolveTCPAddr("tcp", nrps.BindPort)
	l, err := net.ListenTCP("tcp", tcpAddress)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	for {

		conn, err := l.AcceptTCP()
		if err != nil {
			panic(err)
		}

		if clientConnect != nil {
			clientConnect.Close()
		}
		clientConnect = conn

		fmt.Printf("新客户端，远程地址：%v，本地地址：%v", clientConnect.RemoteAddr(), clientConnect.LocalAddr())

		go nrpClientRead()
	}

}
func nrpClientReadPack(b []byte) {

	var id uint64
	var lenght int32
	readBuffer := bytes.NewBuffer(b)
	binary.Read(readBuffer, binary.LittleEndian, &id)
	binary.Read(readBuffer, binary.LittleEndian, &lenght)
	bb := make([]byte, lenght)
	binary.Read(readBuffer, binary.LittleEndian, &bb)
	//fmt.Println("--s-s------")
	//fmt.Println(string(bb))

	fmt.Println("-----------id------------")
	fmt.Println(id)
	h,ok:=outList.Get(id)
	if ok{
		h.out <- bb
		close(h.out)
	}

}
func nrpClientRead() {

	defer func() {
		if clientConnect != nil {
			clientConnect.Close()
		}
	}()
	buf := make([]byte, 0)

	var buffer [4096]byte
	for {

		n, err := clientConnect.Read(buffer[0:])

		if err != nil {

			return
		}
		if n == 0 {
			continue
		}

		//fmt.Println(string(buffer[:n]))

		buf = append(buf, buffer[:n]...)

		for {
			if len(buf) >= 12 {
				testtH := buf[0:12]
				var id uint64
				var lenght int32
				readBuffer := bytes.NewBuffer(testtH)
				binary.Read(readBuffer, binary.LittleEndian, &id)
				binary.Read(readBuffer, binary.LittleEndian, &lenght)

				if int32(len(buf)) >= lenght+12 {

					packs := buf[0 : lenght+12]
					buf = append(make([]byte, 0), buf[lenght+12:]...)
					go nrpClientReadPack(packs)
					if len(buf) > 0 {
						fmt.Printf("还有%v数据\n", len(buf))
					}
				} else {
					break
				}
			} else {
				break
			}
		}

		//fmt.Println(err)
	}

}
