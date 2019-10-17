package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/nbvghost/glog"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

type Nrpc struct {
	ServerIp string
	LocalIp  string
}

var nrpc Nrpc
var conn net.Conn
var ConfigFile string

func init() {

	flag.StringVar(&ConfigFile, "config", "nrpc.json", "-config nrpc.json")
	flag.Parse()

	log.SetFlags(log.Lshortfile)

	glog.Param.Debug = true
	//glog.Param.ServerAddr = ""
	glog.Param.FileStorage = true
	glog.Param.ServerName = "NRPc"
	glog.Param.LogFilePath = "log"
	glog.StartLogger(glog.Param)

}
func main() {

	b, err := ioutil.ReadFile(ConfigFile)
	if glog.Error(err) {
		panic(err)
	}
	json.Unmarshal(b, &nrpc)

	//fmt.Println(nrpc)

	go func() {

		for {
			if conn != nil {
				lenght := int32(0)
				id := uint64(0)
				var b []byte
				buffer := bytes.NewBuffer(make([]byte, 0))
				binary.Write(buffer, binary.LittleEndian, &id)     //8
				binary.Write(buffer, binary.LittleEndian, &lenght) //4
				binary.Write(buffer, binary.LittleEndian, &b)
				dfs := buffer.Bytes()
				_, err = conn.Write(dfs)
				glog.Error(err)
			}

			time.Sleep(time.Second * 3)
		}

	}()

	for {
		conn, err = net.Dial("tcp", nrpc.ServerIp)
		if glog.Error(err) {
			glog.Trace("无法链接到服务器")
			time.Sleep(3 * time.Second)
			glog.Trace("正在重链接")
			continue
		}

		glog.Trace(fmt.Sprintf("已经连接到服务器,本地地址：%v,远程地址：%v\n", conn.LocalAddr(), conn.RemoteAddr()))

		readNrpsData()

		time.Sleep(3 * time.Second)

	}

	//fmt.Println(conn)
	//fmt.Println(httpconn)

}

func readPack(b []byte) {

	var id uint64
	var lenght int32
	readBuffer := bytes.NewBuffer(b)
	binary.Read(readBuffer, binary.LittleEndian, &id)
	binary.Read(readBuffer, binary.LittleEndian, &lenght)
	bb := make([]byte, lenght)
	binary.Read(readBuffer, binary.LittleEndian, &bb)
	//fmt.Printf(string(bb))

	if len(bb) == 0 {
		return
	}

	//glog.Trace("-----------id------------")
	//glog.Trace(id)
	//glog.Trace(string(bb))

	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(bb)))
	if glog.Error(err) {

		return
	}
	fdfs, err := httputil.DumpRequest(req, true)
	glog.Error(err)
	//fmt.Println(string(fdfs))

	tcpaddr, err := net.ResolveTCPAddr("tcp", nrpc.LocalIp)
	if glog.Error(err) {

		return
	}
	httpconn, err := net.DialTCP("tcp", nil, tcpaddr)
	if glog.Error(err) {

		return
	}
	_, err = httpconn.Write(fdfs)
	if glog.Error(err) {

		return
	}
	req.Host = "www.baidu.com:80"
	resp, err := http.ReadResponse(bufio.NewReader(httpconn), req)
	if glog.Error(err) {
		return
	}

	sdf, err := httputil.DumpResponse(resp, true)
	if glog.Error(err) {
		return
	}
	//fmt.Println(string(sdf))

	err = makePackage(id, sdf)
	if glog.Error(err) {
		glog.Trace("-------------写入代理-------------------", err.Error())
	} else {
		glog.Trace("-------------写入代理-------------------")
	}

}
func makePackage(id uint64, body []byte) error {
	lenght := int32(len(body))
	buffer := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buffer, binary.LittleEndian, &id)     //8
	binary.Write(buffer, binary.LittleEndian, &lenght) //4
	binary.Write(buffer, binary.LittleEndian, &body)

	dfs := buffer.Bytes()

	_, err := conn.Write(dfs)
	if glog.Error(err) {
		return err
	}
	return nil
}
func readNrpsData() {

	buf := make([]byte, 0)

	for {
		tempBuf := make([]byte, 4096)
		n, err := conn.Read(tempBuf)
		if glog.Error(err) {
			return
		}
		if n == 0 {
			continue
		}

		buf = append(buf, tempBuf[:n]...)

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

					if len(buf)>0 {
						glog.Trace(fmt.Sprintf("分包处理数据长度：%v", len(buf)))
					}

				} else {
					break
				}
			} else {
				break
			}
		}
		//fmt.Println(n)
		//fmt.Println(err)
		//fmt.Println(string(buf))
		//httpconn.Write(buf[:n])

	}

}
