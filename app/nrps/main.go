package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/nbvghost/glog"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"runtime/debug"
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
func main() {

	glog.Param.Debug =true
	//glog.Param.ServerAddr = ""
	glog.Param.FileStorage = true
	glog.Param.ServerName = "NRPs"
	glog.Param.LogFilePath = "log"
	glog.StartLogger(glog.Param)


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


	glog.Trace("start server at",tcpAddress.Network(),tcpAddress.String())
	glog.Trace("listen at",l.Addr().Network(),l.Addr().String())

	go func() {

		for{
			if clientConnect!=nil{
				lenght := int32(0)
				id:=uint64(0)
				dfs:=make([]byte,0)
				buffer := bytes.NewBuffer(make([]byte, 0))
				binary.Write(buffer, binary.LittleEndian, &id)     //8
				binary.Write(buffer, binary.LittleEndian, &lenght) //4
				binary.Write(buffer, binary.LittleEndian, &dfs)
				_,err:=clientConnect.Write(buffer.Bytes())
				if glog.Error(err){

				}
			}
			time.Sleep(1*time.Second)
		}

	}()

	for {

		conn, err := l.AcceptTCP()
		if err != nil {
			panic(err)
		}
		conn.SetKeepAlive(true)
		if clientConnect != nil {
			clientConnect.Close()
		}
		clientConnect = conn

		glog.Trace(fmt.Sprintf("新客户端，远程地址：%v，本地地址：%v", clientConnect.RemoteAddr(), clientConnect.LocalAddr()))


		go func() {
			nrpClientRead()
		}()

	}

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
	glog.Error(err)
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
			glog.Error(err)
		}()
		_, err = io.Copy(conn, server)
		glog.Error(err)

	} else {
		dfs, err := httputil.DumpRequest(resp, true)
		//fmt.Println(string(dfs))
		glog.Error(err)

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

			_,err:=clientConnect.Write(buffer.Bytes())
			if glog.Error(err){

			}


			glog.Trace("数据写给客户端，等待客户端返回")

			h,ok:=outList.Get(id)
			if ok{
				select {
				case bdfd := <-h.out:
					conn.Write(bdfd)
					//close(hp.out)
					glog.Trace("-----------输出数据-----------------")
				case <-time.After(10*time.Second):
					conn.Write([]byte("timeout"))
					//close(hp.out)
					glog.Trace("-----------客户端返回超时-----------------")

				}
			}else{
				conn.Write([]byte("数据出错"))
				//close(hp.out)
				glog.Trace("-----------数据出错-----------------")
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

	//fmt.Println("-----------id------------")
	//fmt.Println(id)
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

				packLen:=lenght+12
				glog.Trace("nrpClientRead",packLen,len(buf))
				if int32(len(buf)) >= packLen && packLen>0 {
					packs := buf[0 : packLen]
					buf = append(make([]byte, 0), buf[lenght+12:]...)



					go func(args []byte) {
						defer func() {
							if r := recover(); r != nil {
								b := debug.Stack()
								glog.Debug(r, b)
							}
						}()
						nrpClientReadPack(args)
					}(packs)


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
