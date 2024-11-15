# nrp
golang 内网穿透 反向代理 内网公网访问 http协议

目前只支持http协议

服务端：
src/nrps
配制（nrps.json）：
{
  "BindPort":":7070",//服务器端口
  "HttpPort":":8040"//外网访问端口
}


客户端：
src/nrpc
配制（nrpc.json）：
{
  "ServerIp":"nutsy.cc:7070",//服务器地址
  "LocalIp":"127.0.0.1:80"//本地服务地址
}


