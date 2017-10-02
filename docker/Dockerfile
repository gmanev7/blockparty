FROM ubuntu:latest
LABEL Description="Dell Blockparty Web App (https://github.com/seethedata/blockparty) backed by locally deployed Ethereum implementation"

RUN apt-get update && apt-get -y install software-properties-common
RUN add-apt-repository -y ppa:gophers/archive
RUN apt-get update && apt-get -y install golang-1.9-go

WORKDIR /root/adl

COPY *.go .

ENV PATH "$PATH:/usr/lib/go-1.9/bin"
RUN go build blockparty.go
#RUN mkdir data && mkdir source

EXPOSE 8080
