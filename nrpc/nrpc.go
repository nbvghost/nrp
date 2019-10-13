package main

import (
	"github.com/nbvghost/glog"
	"github.com/nbvghost/gweb/tool"
	"github.com/nbvghost/nrp/remote/desktop/windows"
	"net"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"bytes"
	"encoding/binary"
	"net/http"
	"bufio"
	"net/http/httputil"
	"net/url"
	"os"
	"runtime"
	"log"
	"strings"
	"time"
	"flag"
)
type Nrpc struct {
	ServerIp string
	LocalIp string
	ProxyPort string
}
var nrpc Nrpc
var conn net.Conn

var ConfigFile string
var U string
var P string

func init()  {

	flag.StringVar(&ConfigFile, "config", "nrpc.json", "-config nrpc.json")
	flag.StringVar(&U, "u", "", "-u user")
	flag.StringVar(&P, "p", "", "-p password")
	flag.Parse()

	fmt.Println(U,P)

	if strings.EqualFold(U,"") || strings.EqualFold(P,""){
		log.Panic("u and p is not nil")
		os.Exit(0)
	}

}
func main() {


	win:=&windows.TargetServer{}


	loginParam:=url.Values{}
	loginParam.Set("email",U)
	loginParam.Set("password",P)

	req,err:=http.NewRequest("GET","http://127.0.0.1:8055/account/targetLogin?"+loginParam.Encode(),nil)
	glog.Error(err)

	session := &http.Cookie{Name: "GLSESSIONID", Value: win.GetSessionKey(), Path: "/"}
	req.AddCookie(session)

	resp,errd:=http.DefaultClient.Do(req)
	glog.Error(errd)
	b,errd:=ioutil.ReadAll(resp.Body)
	glog.Error(errd)

	glog.Trace(string(b))


	result:=make(map[string]interface{})
	tool.JsonUnmarshal(b,&result)



	b,err=ioutil.ReadFile(ConfigFile)
	if err!=nil{
		panic(err)
	}

	json.Unmarshal(b,&nrpc)





	go win.Start(result["Data"].(map[string]interface{})["ProxyHost"].(string))


	//glog.Trace(nrpc)
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
	//glog.Trace(conn)
	//glog.Trace(httpconn)
	fmt.Printf("已经连接到服务器,本地地址：%v,远程地址：%v\n",conn.LocalAddr(),conn.RemoteAddr())
	go func() {

		for{
			//发送心跳包
			//lenght :=int32(0)
			id :=uint64(0)
			var b []byte
			writePackage(b,id)

			/*buffer:=bytes.NewBuffer(make([]byte,0))
			binary.Write(buffer,binary.LittleEndian,&id)//8
			binary.Write(buffer,binary.LittleEndian,&lenght)//4
			binary.Write(buffer,binary.LittleEndian,&b)
			dfs:=buffer.Bytes()
			_,err=conn.Write(dfs)
			CheckError(err)*/

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
	fmt.Printf(string(bb))


	req,err:=http.ReadRequest(bufio.NewReader(bytes.NewReader(bb)))
	CheckError(err)
	fdfs,err:=httputil.DumpRequest(req,true)
	glog.Trace(string(fdfs))


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
	//glog.Trace(string(sdf))

	writePackage(sdf,id)
}
func writePackage(packages []byte,id uint64)(n int, err error)  {

	lenght :=int32(len(packages))
	buffer:=bytes.NewBuffer(make([]byte,0))
	binary.Write(buffer,binary.LittleEndian,&id)//8
	binary.Write(buffer,binary.LittleEndian,&lenght)//4
	binary.Write(buffer,binary.LittleEndian,&packages)
	dfs:=buffer.Bytes()

	_,err=conn.Write(dfs)
	CheckError(err)
	glog.Trace("-------------写入代理-------------------")

	return 0,nil
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

	}

}
