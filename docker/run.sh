#/bin/sh
#TODO expose the http port to the host network interfaces. '-p' should have worked
#port forwarding setup fails with:
#docker: Error response from daemon: driver failed programming external connectivity on endpoint blockparty (aa906622c17a6767aaca380b1552b0b685132cbbaf9c72c6cbf48b11e60d0a2c):  (iptables failed: iptables --wait -t nat -A DOCKER -p tcp -d 0/0 --dport 32787 -j DNAT --to-destination 172.17.0.2:8080 ! -i docker0: iptables: No chain/target/match by that name.

docker run -p 8080:8080 -p 8545:8545 --name blockparty adl/dell-demo-ui:0.1
#docker run --name blockparty -i -t adl/dell-demo-ui:0.1
