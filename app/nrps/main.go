package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/nbvghost/nrp/domain/httppack"

	"github.com/nbvghost/nrp/domain/unpack"
	"github.com/nbvghost/nrp/model"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

var outList httppack.OutList
var globalID uint64
var nrpS model.NrpS
var clientConnect *net.TCPConn

func init() {
	log.SetFlags(log.Lshortfile)

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
	b, err := ioutil.ReadFile("nrps.json")
	if err != nil {
		panic(err)
	}

	json.Unmarshal(b, &nrpS)

	//fmt.Println(nrps)

	tcpAddress, err := net.ResolveTCPAddr("tcp", nrpS.ServerPort)
	l, err := net.ListenTCP("tcp", tcpAddress)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	log.Println("start server at", tcpAddress.Network(), tcpAddress.String())
	log.Println("listen at", l.Addr().Network(), l.Addr().String())

	go func() {

		for {
			if clientConnect != nil {
				length := int32(0)
				id := uint64(0)
				body := make([]byte, 0)

				//buffer := bytes.NewBuffer(make([]byte, 0))
				//binary.Write(buffer, binary.LittleEndian, &id)     //8
				//binary.Write(buffer, binary.LittleEndian, &lenght) //4
				//binary.Write(buffer, binary.LittleEndian, &dfs)
				buffer, err := unpack.ToBytes(unpack.NewHead(id, length, unpack.HeadTypeHeartbeat), body)
				if err != nil {
					log.Println(err)
					continue
				}
				_, err = clientConnect.Write(buffer.Bytes())
				if err != nil {
					log.Println("写入客户端心跳包失败")
				}
			}
			time.Sleep(1 * time.Second)
		}

	}()

	for {

		clientConnect, err = l.AcceptTCP()
		if err != nil {
			panic(err)
		}
		err = clientConnect.SetKeepAlive(true)
		if err != nil {
			panic(err)
		}

		log.Println(fmt.Sprintf("新客户端，远程地址：%v，本地地址：%v", clientConnect.RemoteAddr(), clientConnect.LocalAddr()))

		go func() {
			nrpClientRead()
		}()

	}

}

var httpListen net.Listener

func startHttp() {
	var err error

	if httpListen != nil {
		httpListen.Close()
	}

	httpListen, err = net.Listen("tcp", nrpS.ProxyPort)
	if err != nil {
		panic(err)
	}
	defer httpListen.Close()

	for {
		conn, err := httpListen.Accept()
		if err != nil {
			return
		}
		go readWeb(conn)
	}
}
func startTCP() {
	var conn net.Conn
	var err error

	var closeChan = make(chan int)
	defer func() {
		closeChan <- 1
	}()
	go func() {
		var id uint64 = 0
		hp := &httppack.HttpPack{Out: make(chan []byte), Time: time.Now()}
		outList.Set(id, hp)

		h, ok := outList.Get(id)

		if ok {
			for {
				select {
				case body := <-h.Out:
					_, err := conn.Write([]byte("body"))
					_, err = conn.Write(body)
					if err != nil {
						log.Println(err)
					}
					//close(hp.out)
					//log.Println("-----------输出数据-----------------")
				case <-closeChan:
					return

				}
			}
		} else {

		}
	}()

	l, err := net.Listen("tcp", nrpS.ProxyPort)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	for {
		conn, err = l.Accept()
		if err != nil {
			panic(err)
		}

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

			buffer, err := unpack.ToBytes(unpack.NewHead(0, int32(len(body)), unpack.HeadTypeData), body)
			if err != nil {
				continue
			}
			clientConnect.Write(buffer.Bytes())

		}
	}
}

var lock sync.RWMutex

func readWeb(conn net.Conn) {
	defer conn.Close()
	//fmt.Println(conn.LocalAddr())
	reader := bufio.NewReader(conn)

	resp, err := http.ReadRequest(reader)
	if err != nil {
		log.Println(err)
	}
	if resp == nil {
		//fmt.Println(resp)
		return
	}

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
			log.Println(err)
		}()
		_, err = io.Copy(conn, server)
		log.Println(err)

	} else {
		body, err := httputil.DumpRequest(resp, true)
		//fmt.Println(string(dfs))
		if err != nil {
			log.Println(err)
		}

		if clientConnect != nil {
			//clientConnect.Write(b)
			lock.Lock()
			var id = globalID + 1
			globalID = id
			lock.Unlock()

			length := int32(len(body))
			/*buffer := bytes.NewBuffer(make([]byte, 0))
			binary.Write(buffer, binary.LittleEndian, &id)     //8
			binary.Write(buffer, binary.LittleEndian, &lenght) //4
			binary.Write(buffer, binary.LittleEndian, &dfs)*/

			buffer, err := unpack.ToBytes(unpack.NewHead(id, length, unpack.HeadTypeData), body)
			if err != nil {
				return
			}

			//fmt.Println(len(dfs))
			hp := &httppack.HttpPack{Out: make(chan []byte), Time: time.Now()}
			outList.Set(id, hp)

			_, err = clientConnect.Write(buffer.Bytes())
			if err != nil {
				log.Println(err)
				return
			}

			//log.Println("数据写给客户端，等待客户端返回")

			h, ok := outList.Get(id)
			if ok {
				select {
				case bdfd := <-h.Out:
					conn.Write(bdfd)
					//close(hp.out)
					//log.Println("-----------输出数据-----------------")
				case <-time.After(10 * time.Second):
					conn.Write([]byte("timeout"))
					//close(hp.out)
					//log.Println("-----------客户端返回超时-----------------")

				}
			} else {
				conn.Write([]byte("数据出错"))
				//close(hp.out)
				log.Println("-----------数据出错-----------------")
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

func nrpClientReadPack(pack []byte) {

	//var id uint64
	//var lenght int32
	//readBuffer := bytes.NewBuffer(pack)
	//binary.Read(readBuffer, binary.LittleEndian, &id)
	//binary.Read(readBuffer, binary.LittleEndian, &lenght)

	head, err := unpack.FromBytes(pack)
	if err != nil {
		return
	}
	body := pack[unpack.HeadLen():]

	log.Println("nrps收到客户的信息：" + string(body))

	//bb := make([]byte, length)
	//binary.Read(readBuffer, binary.LittleEndian, &bb)
	//fmt.Println("--s-s------")
	//fmt.Println(string(bb))

	//fmt.Println("-----------id------------")
	//fmt.Println(id)

	if head.Type == unpack.HeadTypeData {
		h, ok := outList.Get(head.ID)
		if ok {
			h.Out <- body
			if requestType == unpack.HeadTypeHTTP {
				close(h.Out)
			}

		}
	} else if head.Type == unpack.HeadTypeHTTP {
		requestType = unpack.HeadTypeHTTP
		go startHttp()

	} else if head.Type == unpack.HeadTypeTCP {
		requestType = unpack.HeadTypeTCP
		go startTCP()
	}
}

var requestType unpack.HeadType

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
			if len(buf) >= unpack.HeadLen() {
				//testtH := buf[0:12]
				//var id uint64
				//var lenght int32

				//readBuffer := bytes.NewBuffer(testtH)
				//binary.Read(readBuffer, binary.LittleEndian, &id)
				//binary.Read(readBuffer, binary.LittleEndian, &lenght)

				head, err := unpack.FromBytes(buf)
				if err != nil {
					break
				}

				packLen := head.Length + int32(unpack.HeadLen())
				//glog.Trace("nrpClientRead",packLen,len(buf))
				if int32(len(buf)) >= packLen && packLen > 0 {
					packs := buf[0:packLen]
					buf = append(make([]byte, 0), buf[packLen:]...)

					go func(args []byte) {
						defer func() {
							if r := recover(); r != nil {
								b := debug.Stack()
								log.Println(r, b)
							}
						}()
						nrpClientReadPack(args)
					}(packs)

					if len(buf) > 0 {
						//fmt.Printf("还有%v数据\n", len(buf))
						log.Println(fmt.Sprintf("分包处理数据长度：%v", len(buf)))
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
