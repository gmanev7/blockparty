FROM ubuntu:latest
LABEL Description="Building the Dell Blockparty Web App"

RUN apt-get update && apt-get -y install software-properties-common
RUN add-apt-repository -y ppa:gophers/archive

RUN apt-get update && apt-get -y install golang-1.9-go
RUN apt-get -y install git # necessary for go get to download source code from remote git repositories
WORKDIR /root/adl

ENV PATH "$PATH:/usr/lib/go-1.9/bin"
#TODO use https://github.com/tools/godep to install the dependencies with 'godep restore'
RUN go get -v -u github.com/garyburd/redigo/redis
RUN go get -v -u github.com/gorilla/mux
RUN go get -v -u github.com/gorilla/sessions
RUN go get -v -u github.com/rs/cors
RUN go get -v -u github.com/satori/go.uuid

COPY *.go .
RUN go build blockparty.go

##################################

FROM localhost:5000/adl/dell-demo:latest
LABEL Description="Dell Blockparty Web App (https://github.com/seethedata/blockparty) backed by locally deployed Ethereum implementation"

RUN apt-get -y install redis-server
COPY --from=0 /root/adl/blockparty .

COPY houses.json .
COPY users.json .
COPY addresses.json .
COPY css /app/css/
COPY fonts /app/fonts/
COPY images /app/images/
COPY js /app/js/
COPY templates templates/
COPY launch.sh .

RUN systemctl enable redis-server
EXPOSE 8080

ENTRYPOINT ["/bin/bash", "launch.sh"]
