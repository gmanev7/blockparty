FROM ubuntu:latest
LABEL Description="Dell Blockparty Web App (https://github.com/seethedata/blockparty) backed by locally deployed Ethereum implementation"

RUN apt-get update && apt-get -y install software-properties-common
RUN add-apt-repository -y ppa:gophers/archive
RUN apt-get update && apt-get -y install golang-1.9-go
RUN apt-get -y install git # necessary for go get to download source code from remote git repositories
WORKDIR /root/adl

ENV PATH "$PATH:/usr/lib/go-1.9/bin"
RUN go get -v -u github.com/cloudfoundry-community/go-cfenv
RUN go get -v -u github.com/garyburd/redigo/redis
RUN go get -v -u github.com/gorilla/mux
RUN go get -v -u github.com/gorilla/sessions
RUN go get -v -u github.com/satori/go.uuid

COPY *.go .
COPY services.json .

RUN go build blockparty.go

EXPOSE 8080
