#!/usr/bin/env sh
openssl req \
    -new \
    -newkey rsa:4096 \
    -days 365 \
    -nodes \
    -x509 \
    -subj "/C=US/ST=Denial/L=DockerLand/O=Dis/CN=flashpaper" \
    -keyout ./server.key \
    -out ./server.crt

./go-flashpaper
