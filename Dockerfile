FROM golang:1.7 AS build

WORKDIR /go/src/github.com/Invoca/go-flashpaper
COPY *.go .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo

FROM alpine
WORKDIR /
RUN apk update && apk add openssl

COPY --from=build /go/src/github.com/Invoca/go-flashpaper/go-flashpaper .
COPY ./entrypoint.sh .

EXPOSE 8443
ENTRYPOINT ["./entrypoint.sh"]
