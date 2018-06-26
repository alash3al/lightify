FROM golang:alpine

RUN apk update && apk add git

RUN go get github.com/alash3al/lightify

ENTRYPOINT ["lightify"]

WORKDIR /root/