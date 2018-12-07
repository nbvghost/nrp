package main

import (
	"net"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"bytes"
	"encoding/binary"
	"net/http"
	"bufio"
	"net/http/httputil"
	"runtime"
	"log"
	"time"
	"flag"
)
type Nrpc struct {
	ServerIp string
	LocalIp string
}
var nrpc Nrpc
var conn net.Conn
var ConfigFile string
func init()  {

	flag.StringVar(&ConfigFile, "config", "nrpc.json", "-config nrpc.json")
	flag.Parse()
}
func main() {

	b,err:=ioutil.ReadFile(ConfigFile)
	if err!=nil{
		panic(err)
	}


	json.Unmarshal(b,&nrpc)


	//fmt.Println(nrpc)


	conn,err=net.Dial("tcp",nrpc.ServerIp)
	if err!=nil{

		for{
			conn,err=net.Dial("tcp",nrpc.ServerIp)
			if err==nil{
				break
			}
			log.Println("无法链接到服务器")
			time.Sleep(1*time.Second)
		}


	}
	//fmt.Println(conn)
	//fmt.Println(httpconn)
	fmt.Printf("已经连接到服务器,本地地址：%v,远程地址：%v\n",conn.LocalAddr(),conn.RemoteAddr())
	go func() {

		for{

			lenght :=int32(0)
			id :=uint64(0)
			var b []byte
			buffer:=bytes.NewBuffer(make([]byte,0))
			binary.Write(buffer,binary.LittleEndian,&id)//8
			binary.Write(buffer,binary.LittleEndian,&lenght)//4
			binary.Write(buffer,binary.LittleEndian,&b)
			dfs:=buffer.Bytes()
			_,err=conn.Write(dfs)
			CheckError(err)

			time.Sleep(time.Second*3)
		}

	}()
	read()

}
func CheckError(err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		log.Println(file, line, err)
	}
}

func readPack(b []byte)  {

	var id uint64
	var lenght int32
	readBuffer:=bytes.NewBuffer(b)
	binary.Read(readBuffer,binary.LittleEndian,&id)
	binary.Read(readBuffer,binary.LittleEndian,&lenght)
	bb:=make([]byte,lenght)
	binary.Read(readBuffer,binary.LittleEndian,&bb)
	//fmt.Printf(string(bb))



	fmt.Println("-----------id------------")
	fmt.Println(id)

	req,err:=http.ReadRequest(bufio.NewReader(bytes.NewReader(bb)))
	CheckError(err)
	fdfs,err:=httputil.DumpRequest(req,true)
	//fmt.Println(string(fdfs))


	tcpaddr, err := net.ResolveTCPAddr("tcp", nrpc.LocalIp)
	if err!=nil{
		return
	}
	httpconn,err:=net.DialTCP("tcp",nil,tcpaddr)
	if err!=nil{
		return
	}
	_,err=httpconn.Write(fdfs)
	if err!=nil{
		return
	}
	resp,err:=http.ReadResponse(bufio.NewReader(httpconn),req)
	if err!=nil{
		return
	}
	sdf,err:=httputil.DumpResponse(resp,true)
	if err!=nil{
		return
	}
	//fmt.Println(string(sdf))



	lenght =int32(len(sdf))
	buffer:=bytes.NewBuffer(make([]byte,0))
	binary.Write(buffer,binary.LittleEndian,&id)//8
	binary.Write(buffer,binary.LittleEndian,&lenght)//4
	binary.Write(buffer,binary.LittleEndian,&sdf)

	dfs:=buffer.Bytes()

	_,err=conn.Write(dfs)
	if err!=nil{
		return
	}
	fmt.Println("-------------写入代理-------------------")

}
func read()  {


	buf := make([]byte,0)

	for{
		tempBuf:=make([]byte,4096)
		n,err:= conn.Read(tempBuf)
		if err!=nil{
			return
		}
		if n==0{
			continue
		}

		buf=append(buf, tempBuf[:n]...)

		for{
			if len(buf)>12{

				testtH:=buf[0:12]

				var id uint64
				var lenght int32
				readBuffer:=bytes.NewBuffer(testtH)
				binary.Read(readBuffer,binary.LittleEndian,&id)
				binary.Read(readBuffer,binary.LittleEndian,&lenght)

				if int32(len(buf))>=lenght+12{

					packs:=buf[0:lenght+12]
					buf=append(make([]byte,0), buf[lenght+12:]...)
					go readPack(packs)
					fmt.Printf("还有%v数据\n",len(buf))

				}else{
					break
				}
			}else{
				break
			}
		}


		//fmt.Println(n)
		//fmt.Println(err)
		//fmt.Println(string(buf))

		//httpconn.Write(buf[:n])

	}

}
