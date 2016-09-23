FROM golang:1.7.1-onbuild
MAINTAINER Jimmy Mesta "jimmy.mesta@gmail.com"

RUN wget https://dl.eff.org/certbot-auto -P /usr/local/sbin && \
    chmod a+x /usr/local/sbin/certbot-auto && \
    git clone https://github.com/rawdigits/go-flashpaper

WORKDIR go-flashpaper
RUN go build

RUN openssl req \
    -new \
    -newkey rsa:4096 \
    -days 365 \
    -nodes \
    -x509 \
    -subj "/C=US/ST=Denial/L=DockerLand/O=Dis/CN=www.flashpaper.com" \
    -keyout /go/src/app/go-flashpaper/server.key \
    -out /go/src/app/go-flashpaper/server.crt

EXPOSE 8443

ENTRYPOINT ["./go-flashpaper"] 
