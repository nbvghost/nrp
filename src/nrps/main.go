package main

import (
	"io/ioutil"
	"fmt"
	"encoding/json"
	"net"
	"bytes"
	"encoding/binary"
	"net/http"
	"bufio"
	"runtime"
	"log"
	"net/http/httputil"
	"time"
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
	out chan []byte
	time time.Time
}

func init()  {
	outList = make(map[uint64]HttpPack)

	go func() {

		for _,value:=range outList{
			//outList
			if time.Now().Unix()-value.time.Unix()>10{
				close(value.out)
			}
		}

	}()
}

func startWeb()  {
	l,err:=net.Listen("tcp",nrps.HttpPort)
	if err!=nil{
		panic(err)
	}
	defer l.Close()

	for {

		conn,err:=l.Accept()
		if err!=nil{
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

func readweb(conn net.Conn)  {

	//fmt.Println(conn.LocalAddr())

	resp,err:=http.ReadRequest(bufio.NewReader(conn))
	CheckError(err)

	dfs,err:=httputil.DumpRequest(resp,true)
	CheckError(err)
	defer conn.Close()

	if clientConnect!=nil{
		//clientConnect.Write(b)

		id=id+1
		lenght :=int32(len(dfs))
		buffer:=bytes.NewBuffer(make([]byte,0))
		binary.Write(buffer,binary.LittleEndian,&id)//8
		binary.Write(buffer,binary.LittleEndian,&lenght)//4
		binary.Write(buffer,binary.LittleEndian,&dfs)

		//fmt.Println(len(dfs))
		hp:=HttpPack{out:make(chan []byte),time:time.Now()}
		outList[id] = hp
		clientConnect.Write(buffer.Bytes())
		bdfd:=<-hp.out

		conn.Write(bdfd)
		close(hp.out)
		fmt.Println("-----------输出数据-----------------")
		//fmt.Println(string(bdfd))
	}
}
func main() {

	b,err:=ioutil.ReadFile("nrps.json")
	if err!=nil{
		panic(err)
	}


	json.Unmarshal(b,&nrps)


	//fmt.Println(nrps)


	go startWeb()


	tcpAddress,err:=net.ResolveTCPAddr("tcp",nrps.BindPort)
	l,err:=net.ListenTCP("tcp",tcpAddress)
	if err!=nil{
		panic(err)
	}
	defer l.Close()

	for {

		conn,err:=l.AcceptTCP()
		if err!=nil{
			panic(err)
		}

		if clientConnect!=nil{
			clientConnect.Close()
		}
		clientConnect = conn

		go read()
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

	//fmt.Println("--s-s------")
	//fmt.Println(string(bb))

	outList[id].out<-bb
}
func read()  {


	defer func() {
		if clientConnect!=nil{
			clientConnect.Close()
		}
	}()
	buf := make([]byte,0)

	var buffer [4096]byte
	for{

		n,err := clientConnect.Read(buffer[0:])

		if err!=nil{

			return
		}
		if n==0{
			continue
		}

		buf=append(buf, buffer[:n]...)

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
					fmt.Printf("还有%v数据",len(buf))
				}else{
					break
				}
			}else{
				break
			}
		}

		//fmt.Println(err)
	}

}
