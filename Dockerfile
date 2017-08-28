FROM golang-1.7 AS build

COPY *.go ./

RUN openssl req \
    -new \
    -newkey rsa:4096 \
    -days 365 \
    -nodes \
    -x509 \
    -subj "/C=US/ST=Denial/L=DockerLand/O=Dis/CN=www.flashpaper.com" \
    -keyout /go/src/app/go-flashpaper/server.key \
    -out /go/src/app/go-flashpaper/server.crt

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo
FROM alpine
WORKDIR /
COPY --from=build /go/src/github.com/Invoca/flashpaper-go/flashpaper-go .

EXPOSE 8443

CMD ["./go-flashpaper"]
