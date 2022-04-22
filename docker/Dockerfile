FROM debian:10

ADD sources.list /etc/apt/sources.list

ENV PATH=/opt/go/bin:$PATH

RUN apt update && apt install -y vim curl mingw-w64 git upx ca-certificates \
	&& cd /opt \
	&& curl -o go.linux-amd64.tar.gz "https://dl.google.com/go/$(curl https://golang.google.cn/VERSION?m=text).linux-amd64.tar.gz" \
	&& tar xf go.linux-amd64.tar.gz && rm -rf go.linux-amd64.tar.gz \
	&& go env -w GOPROXY=https://goproxy.cn,https://goproxy.io,direct

WORKDIR /opt/src
