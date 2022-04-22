# webvnc
在 https://github.com/phoboslab/jsmpeg-vnc 的基础上进行封装，通过websocket代理将原有服务端模式转为客户端模块，然后通过单一的服务端提供页面

同时感谢 https://github.com/xtaci/smux 提供单连接复用API

# 编译方法
为了编译方便，建议在ubuntu下进行跨平台编译，因此需要安装golang编译器及mingw-w64

建议通过docker方式安装编译环境，请将代码clone到本地后，在根路径下，使用`docker build -t mingw-w64:1.0 docker`编译docker镜像（mingw-w64:1.0可修改为自己喜欢的名称）

然后启动容器：`docker run -it --rm -v {代码根路径}:/opt/src mingw-w64:1.0`

如需编译客户端请执行：`cd client`；如需编译服务器请执行：`cd server`，编译方法如下：

编译64位，有命令行窗口
```
go mod tidy && go fmt && CGO_ENABLED=1 GOARCH=amd64 GOOS=windows CC=x86_64-w64-mingw32-gcc go build -ldflags "-w -s"
```

编译64位，无命令行窗口
```
go mod tidy && go fmt && CGO_ENABLED=1 GOARCH=amd64 GOOS=windows CC=x86_64-w64-mingw32-gcc go build -ldflags "-w -s -H windowsgui"
```

编译32位，有命令行窗口
```
go mod tidy && go fmt && CGO_ENABLED=1 GOARCH=386 GOOS=windows CC=i686-w64-mingw32-gcc go build -ldflags "-w -s"
```


编译32位，无命令行窗口
```
go mod tidy && go fmt && CGO_ENABLED=1 GOARCH=386 GOOS=windows CC=i686-w64-mingw32-gcc go build -ldflags "-w -s -H windowsgui"
```

# 客户端执行
客户端执行需要指定服务端地址，如192.168.0.100:8230
```
webvnc_client -s 192.168.0.100:8230
```

# 服务端执行
服务端可直接执行，也可指定客户端连接端口，http端口默认为11080，请通过代码修改
```
webvnc_server -p 8230
```
