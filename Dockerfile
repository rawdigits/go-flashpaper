FROM golang AS build

RUN mkdir -p /usr/go/go-flashpaper
WORKDIR /usr/go/go-flashpaper
COPY *.go .

RUN openssl req \
    -new \
    -newkey rsa:4096 \
    -days 365 \
    -nodes \
    -x509 \
    -subj "/C=US/ST=Denial/L=DockerLand/O=Dis/CN=www.flashpaper.com" \
    -keyout ./server.key \
    -out ./server.crt

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo
FROM alpine
WORKDIR /usr/go/go-flashpaper
CMD /bin/sh
COPY --from=build go-flashpaper .

EXPOSE 8443
CMD ["./go-flashpaper"]
