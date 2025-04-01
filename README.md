# MQTT client for OpenWRT

##


## Prepare Go vars

```
export GOPATH=$HOME/bin/go/
export PATH=$GOPATH/bin:$PATH
```

## Build

```
cd src/mqtt-client
GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -ldflags="-w -s" -o bin/mqtt-clnt
cd -
```

## Configuration

```
export MQTT_BROKER=127.0.0.1
export MQTT_PORT=1883
export MQTT_USER=mqtt-user
export MQTT_PASS=mqtt-pass
export MQTT_TOPIC="routers/my-wrt"
```

## Copy binary to router

`cat src/mqtt-client/bin/mqtt-clnt | gzip  | ssh <USER>@<HOST> 'zcat - > mqtt-clnt'`

## Autostart on openwrt router

Add below content to `/etc/rc.local` file
```
sleep 20s # It requres some time to up all network interfaces
source /root/mqtt.conf
(/root/mqtt-clnt) &
logger "/mqtt-mips-linux Started"
```

# Used articles

- http://www.giuseppeparrello.it/en/net_router_howto_add_cron_jobs.php
- https://medium.com/@johnsercel/asus-router-usb-modem-initial-reliability-hacks-74885a2ff318
