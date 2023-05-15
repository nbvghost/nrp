package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/nbvghost/nrp/domain/unpack"
	"github.com/nbvghost/nrp/model"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

var nrpC model.NrpC
var conn net.Conn
var tcpLocalConn net.Conn
var ConfigFile string

func init() {

	flag.StringVar(&ConfigFile, "config", "nrpc.json", "-config nrpc.json")
	flag.Parse()

	log.SetFlags(log.Lshortfile)
}
func main() {

	b, err := ioutil.ReadFile(ConfigFile)
	if err != nil {
		panic(err)
	}
	json.Unmarshal(b, &nrpC)

	//fmt.Println(nrpc)

	go func() {
		//心跳协程/心跳逻辑
		for {
			if conn != nil {
				length := int32(0)
				id := uint64(0)
				var b []byte

				//buffer := bytes.NewBuffer(make([]byte, 0))
				//binary.Write(buffer, binary.LittleEndian, &id)     //8
				//binary.Write(buffer, binary.LittleEndian, &length) //4
				//binary.Write(buffer, binary.LittleEndian, &b)
				buffer, err := unpack.ToBytes(unpack.NewHead(id, length, unpack.HeadTypeHeartbeat), b)
				if err != nil {
					return
				}
				dfs := buffer.Bytes()
				_, err = conn.Write(dfs)
				if err != nil {
					log.Println(err)
				}
			}

			time.Sleep(time.Second * 3)
		}

	}()

	for {
		conn, err = net.Dial("tcp", nrpC.ServerIp)
		if err != nil {
			log.Println("无法链接到服务器")
			time.Sleep(3 * time.Second)
			log.Println("正在重链接")
			continue
		}

		log.Println(fmt.Sprintf("已经连接到服务器,本地地址：%v,远程地址：%v\n", conn.LocalAddr(), conn.RemoteAddr()))

		var netTypeBuffer *bytes.Buffer
		var err error
		if nrpC.Type == model.NrpTypeHTTP {
			netTypeBuffer, err = unpack.ToBytes(unpack.NewHead(0, 0, unpack.HeadTypeHTTP), nil)
		} else if nrpC.Type == model.NrpTypeTCP {
			netTypeBuffer, err = unpack.ToBytes(unpack.NewHead(0, 0, unpack.HeadTypeTCP), nil)
			if err != nil {
				panic(err)
			}
			go handelLocal()

		} else {
			panic("不知道类型：" + nrpC.Type)
		}
		if err != nil {
			panic(err)
		}

		_, err = conn.Write(netTypeBuffer.Bytes())
		if err != nil {
			panic(err)
		}
		readData()
		time.Sleep(3 * time.Second)

	}

	//fmt.Println(conn)
	//fmt.Println(httpconn)

}
func handelLocal() {
	tcpAddr, err := net.ResolveTCPAddr("tcp", nrpC.LocalIp)
	if err != nil {
		panic(err)
	}
	for {
		tcpLocalConn, err = net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second)
			continue
		}
		break
	}

	for {
		tempBuf := make([]byte, 4096)
		n, err := tcpLocalConn.Read(tempBuf)
		if err != nil {
			return
		}
		if n == 0 {
			continue
		}

		body := tempBuf[:n]
		log.Println("nrpc收到本地服务数据：" + string(body))

		buffer, err := unpack.ToBytes(unpack.NewHead(0, int32(len(body)), unpack.HeadTypeData), body)
		if err != nil {
			continue
		}
		conn.Write(buffer.Bytes())

	}
}

func readData() {

	buf := make([]byte, 0)
	for {
		tempBuf := make([]byte, 4096)
		n, err := conn.Read(tempBuf)
		if err != nil {
			return
		}
		if n == 0 {
			continue
		}

		buf = append(buf, tempBuf[:n]...)

		for {
			if len(buf) >= unpack.HeadLen() {

				head, err := unpack.FromBytes(buf)
				if err != nil {
					break
				}

				packLen := int32(unpack.HeadLen()) + head.Length
				if int32(len(buf)) >= packLen {

					packs := buf[0:packLen]
					buf = append(make([]byte, 0), buf[packLen:]...)

					if nrpC.Type == model.NrpTypeHTTP {
						go readHttpPack(packs)
					} else if nrpC.Type == model.NrpTypeTCP {
						go readTCPPack(packs)
					}

					if len(buf) > 0 {
						log.Println(fmt.Sprintf("分包处理数据长度：%v", len(buf)))
					}

				} else {
					break
				}
			} else {
				break
			}
		}
	}
}
func readTCPPack(pack []byte) {
	_, err := unpack.FromBytes(pack)
	if err != nil {
		log.Println(err)
		return
	}
	body := pack[unpack.HeadLen():]
	if len(body) == 0 {
		return
	}
	tcpLocalConn.Write(body)
}
func readHttpPack(pack []byte) {

	head, err := unpack.FromBytes(pack)
	if err != nil {
		log.Println(err)
		return
	}
	//var id uint64
	//var lenght int32
	//readBuffer := bytes.NewBuffer(b)
	//binary.Read(readBuffer, binary.LittleEndian, &id)
	//binary.Read(readBuffer, binary.LittleEndian, &lenght)
	//bb := make([]byte, lenght)
	//binary.Read(readBuffer, binary.LittleEndian, &bb)
	//fmt.Printf(string(bb))
	body := pack[unpack.HeadLen():]
	if len(body) == 0 {
		return
	}

	//glog.Trace("-----------id------------")
	//glog.Trace(id)
	//glog.Trace(string(bb))

	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(body)))
	if err != nil {

		return
	}
	fdfs, err := httputil.DumpRequest(req, true)
	if err != nil {
		log.Println(err)
		return
	}
	//fmt.Println(string(fdfs))

	tcpaddr, err := net.ResolveTCPAddr("tcp", nrpC.LocalIp)
	if err != nil {
		log.Println(err)
		return
	}
	httpconn, err := net.DialTCP("tcp", nil, tcpaddr)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = httpconn.Write(fdfs)
	if err != nil {
		log.Println(err)
		return
	}
	req.Host = "www.baidu.com:80"
	resp, err := http.ReadResponse(bufio.NewReader(httpconn), req)
	if err != nil {
		log.Println(err)
		return
	}

	body, err = httputil.DumpResponse(resp, true)
	if err != nil {
		log.Println(err)
		return
	}
	//fmt.Println(string(sdf))
	length := int32(len(body))
	buffer, err := unpack.ToBytes(unpack.NewHead(head.ID, length, unpack.HeadTypeData), body)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = conn.Write(buffer.Bytes())
	if err != nil {
		log.Println(err)
		return
	}
}
