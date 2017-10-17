#/bin/sh
#TODO pass it down to Ethereum
ethPort=8545

./launch_eth.sh
if [ $? -ne 0 ];
then
    echo "Launching Ethereum nodes failed"
    exit 1
fi

echo "Starting Redis ...."
/etc/init.d/redis-server start
echo "Started Redis"

#TODO figure out the IP address of this machine
#172.17.0.2 is by default the IP of the first container on the bridge network - it is currently used by the local Docker registry container

echo "Starting the UI server ..."
./blockparty 172.17.0.3 8080 ${ethPort}
