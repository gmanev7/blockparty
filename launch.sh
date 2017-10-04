#/bin/sh
echo "Starting Redis ...."
/etc/init.d/redis-server start
echo "Started Redis"
echo "Starting the UI server ..."

#TODO figure out the IP address of this machine
#172.17.0.2 is by default the IP of the first container on the bridge network
./blockparty 172.17.0.2 8080
