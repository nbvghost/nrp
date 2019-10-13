package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/nbvghost/glog"
	"github.com/nbvghost/nrp/remote/desktop/windows"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"runtime"
	"strings"
	"time"
	"io"
)

var outList map[uint64]HttpPack
var id uint64
var nrps Nrps
var clientConnect *net.TCPConn

type Nrps struct {
	BindPort string
	HttpPort string
}
type HttpPack struct {
	out  chan []byte
	time time.Time
}

func init() {
	outList = make(map[uint64]HttpPack)

	go func() {

		for _, value := range outList {
			//outList
			if time.Now().Unix()-value.time.Unix() > 10 {
				close(value.out)
			}
		}

	}()
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

func readweb(conn net.Conn) {
	defer conn.Close()
	//glog.Trace(conn.LocalAddr())
	reader := bufio.NewReader(conn)

	resp, err := http.ReadRequest(reader)
	if resp == nil {
		//glog.Trace(resp)
		return
	}
	CheckError(err)
	//glog.Trace(resp.Method)


	if strings.EqualFold(resp.Method, http.MethodConnect) {
		glog.Trace(resp.Host)
		glog.Trace(resp)

		//glog.Trace(resp.Response)
		//glog.Trace(http.ReadResponse(reader,resp))

		server, err := net.DialTimeout("tcp", resp.Host, time.Second*12)

		if err != nil {
			CheckError(err)
			return
		}

		//http.StatusOK

		fmt.Fprint(conn, "HTTP/1.1 200 Connection established\r\n\r\n")


		/*sb,err:=ioutil.ReadAll(server)
		CheckError(err)
		_,err=conn.Write(sb)
		CheckError(err)*/


		go func() {
			_, err := io.Copy(server, conn)
			CheckError(err)
		}()
		 //cb:=make([]byte,32)
		//cb,err=ioutil.ReadAll(server)//io.ReadFull(server,cb)
		_, err = io.Copy(conn, server)
		CheckError(err)
		//conn.Write(cb)
		//io.CopyBuffer()



	} else {
		dfs, err := httputil.DumpRequest(resp, true)
		//glog.Trace(string(dfs))
		CheckError(err)

		if clientConnect != nil {
			//clientConnect.Write(b)

			id = id + 1
			lenght := int32(len(dfs))
			buffer := bytes.NewBuffer(make([]byte, 0))
			binary.Write(buffer, binary.LittleEndian, &id)     //8
			binary.Write(buffer, binary.LittleEndian, &lenght) //4
			binary.Write(buffer, binary.LittleEndian, &dfs)

			//glog.Trace(len(dfs))
			hp := HttpPack{out: make(chan []byte), time: time.Now()}
			outList[id] = hp
			clientConnect.Write(buffer.Bytes())
			bdfd := <-hp.out

			conn.Write(bdfd)
			close(hp.out)
			glog.Trace("-----------输出数据-----------------")
			//glog.Trace(string(bdfd))

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

	//glog.Trace(nrps)

	go startWeb()

	win:=&windows.WindowsServer{}
	go win.Start()

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

		fmt.Printf("新客户端，远程地址：%v，本地地址：%v\n", clientConnect.RemoteAddr(), clientConnect.LocalAddr())

		go read()
	}

}




func readPack(b []byte) {

	var id uint64
	var lenght int32
	readBuffer := bytes.NewBuffer(b)
	binary.Read(readBuffer, binary.LittleEndian, &id)
	binary.Read(readBuffer, binary.LittleEndian, &lenght)
	bb := make([]byte, lenght)
	binary.Read(readBuffer, binary.LittleEndian, &bb)

	//glog.Trace("--s-s------")
	//glog.Trace(string(bb))

	outList[id].out <- bb
}
func read() {

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

		//glog.Trace(string(buffer[:n]))

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
					go readPack(packs)
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

		//glog.Trace(err)
	}

}