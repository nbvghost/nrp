package windows

import (
	"github.com/nbvghost/glog"
	"github.com/nbvghost/gweb/tool"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
)

const udpPackageLen  = 2048

const targetRemoteDesktopPort  = 3389

type TargetServer struct {
	UUID string
}

func (service *TargetServer)GetSessionKey()string  {
	if strings.EqualFold(service.UUID,""){
		service.UUID = tool.UUID()
	}
	return service.UUID
}
func  (service *TargetServer)Start(ProxyHost string)  {




	go service.startUDP(ProxyHost)


	for{
		proxyConn,err:=net.Dial("tcp",ProxyHost)
		if glog.Error(err)==false{

			service.handProxy(proxyConn)
		}
	}

}

func (service *TargetServer)startUDP(ProxyHost string)  {

	//hosts:=strings.Split(ProxyHost,":")

	prAddr, err := net.ResolveUDPAddr("udp", ProxyHost)
	glog.Error(err)

	//plAddr := &net.UDPAddr{IP:net.ParseIP("127.0.0.1"), Port: 7770}
	//prAddr := &net.UDPAddr{IP: net.ParseIP(hosts[0]), Port: proxyPort}


	//tlAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 7510}
	trAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: targetRemoteDesktopPort}


	proxyConn, err := net.DialUDP("udp", nil, prAddr)
	glog.Error(err)
	targetConn, err := net.DialUDP("udp", nil,trAddr)
	glog.Error(err)

	defer proxyConn.Close()
	defer targetConn.Close()

	glog.Trace(targetConn.RemoteAddr(),targetConn.LocalAddr())
	glog.Trace(proxyConn.LocalAddr().String())
	glog.Trace(proxyConn.LocalAddr().Network())
	glog.Trace(targetConn.RemoteAddr().String())
	glog.Trace(targetConn.RemoteAddr().Network())


	b,err:=tool.JsonMarshal(map[string]interface{}{"TargetProxyLocalHost":proxyConn.LocalAddr().String(),"Network":proxyConn.LocalAddr().Network()})

	ex:=tool.CipherEncrypterData(string(b))

	req,err:=http.NewRequest("GET","http://127.0.0.1:8055/server/listen?auth="+ex,nil)
	glog.Error(err)

	session := &http.Cookie{Name: "GLSESSIONID", Value: service.GetSessionKey(), Path: "/"}
	req.AddCookie(session)


	resp,err:=http.DefaultClient.Do(req)
	b,err=ioutil.ReadAll(resp.Body)
	if glog.Error(err){
		return
	}

	glog.Trace(string(b))


	//var _targetAddr *net.UDPAddr
	//var _proxyAddr *net.UDPAddr


	go func() {

		for{
			data := make([]byte, udpPackageLen)
			n,_, err := targetConn.ReadFromUDP(data)
			if glog.Error(err){
				continue
			}
			//_targetAddr =addr
			//_Addr, _Err := net.ResolveUDPAddr("udp", proxyConn.LocalAddr().String())
			//glog.Error(_Err)
			//glog.Trace(_Addr)
			//glog.Trace("client udp 3",n)
			n, err =proxyConn.Write(data[0:n])
			//n, err =proxyConn.WriteToUDP(data[0:n],_proxyAddr)
			if glog.Error(err){
				continue
			}
		}

	}()

	for{
		data := make([]byte, udpPackageLen)
		n,_, err := proxyConn.ReadFromUDP(data)
		//glog.Trace("读取代理数据",n)
		if glog.Error(err){
			continue
		}
		//_proxyAddr=addr
		//glog.Trace(targetConn.RemoteAddr(),targetConn.LocalAddr())
		//n, err =targetConn.WriteToUDP(data[0:n],_targetAddr)
		//_Addr, _Err := net.ResolveUDPAddr("udp", targetConn.RemoteAddr().String())
		//glog.Error(_Err)
		//glog.Trace(_Addr)
		//glog.Trace("client udp 4",n)
		n, err =targetConn.Write(data[0:n])
		if glog.Error(err){
			continue
		}
	}


}
func  (service *TargetServer)handProxy(proxyConn net.Conn)  {
	//todo:这里的127.0.0.1 是不是要改成 0.0.0.0
	connn,errr:=net.Dial("tcp","127.0.0.1:"+strconv.FormatInt(int64(targetRemoteDesktopPort),10))
	glog.Trace(errr)
	go func() {
		for{
			b:=make([]byte,64)
			n,err:=connn.Read(b)
			//glog.Trace(n,err,b)
			if glog.Error(err){
				proxyConn.Close()
				return
				//return
				//connn,errr=net.Dial("tcp","127.0.0.1:3389")
				//glog.Trace(errr)
				//continue
			}
			n,err=proxyConn.Write(b[0:n])
			//glog.Trace(n,err)

		}
	}()
	for{
		b:=make([]byte,64)
		n,err:=proxyConn.Read(b)
		if glog.Error(err){
			connn.Close()
			return
		}
		//glog.Trace(n,err,b)
		n,err=connn.Write(b[0:n])
		//glog.Trace(n,err)

	}
}
