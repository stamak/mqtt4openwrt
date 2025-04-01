# MQTT Client for OpenWRT

A lightweight MQTT client written in Go and designed for OpenWRT-based routers. This client allows routers to publish messages to an MQTT broker, enabling real-time data exchange for smart home automation (e.g., Home Assistant). Currently, it collects and calculates metrics such as download and upload speed, memory consumption, and CPU utilization. Additionally, it can be extended via configuration or code modifications to report other system metrics or custom data as needed.

## Prerequisites

Ensure you have Go installed and configured properly. Set up your Go environment variables:

```sh
export GOPATH=$HOME/bin/go/
export PATH=$GOPATH/bin:$PATH
```

## Building the MQTT Client

To build the MQTT client for OpenWRT (MIPS architecture), run:

```sh
cd src/mqtt-client
GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -ldflags="-w -s" -o bin/mqtt-clnt
cd -
```

## Configuration

Set the required environment variables before running the client:

```sh
export MQTT_BROKER=127.0.0.1
export MQTT_PORT=1883
export MQTT_USER=mqtt-user
export MQTT_PASS=mqtt-pass
export MQTT_TOPIC="routers/my-wrt"
```

## Deploying to the OpenWRT Router

Transfer the compiled binary to your OpenWRT router:

```sh
cat src/mqtt-client/bin/mqtt-clnt | gzip | ssh <USER>@<HOST> 'zcat - > mqtt-clnt'
```

## Enabling Auto-Start on OpenWRT

To start the MQTT client automatically on boot, add the following lines to `/etc/rc.local`:

```sh
sleep 20s # Allow time for network interfaces to initialize
source /root/mqtt.conf
(/root/mqtt-clnt) &
logger "MQTT client started"
```

## References

For further reading and setup references:

- [Adding cron jobs to OpenWRT](http://www.giuseppeparrello.it/en/net_router_howto_add_cron_jobs.php)
- [ASUS Router USB Modem Reliability Hacks](https://medium.com/@johnsercel/asus-router-usb-modem-initial-reliability-hacks-74885a2ff318)

---

This project simplifies MQTT integration for OpenWRT-based routers, ensuring reliable message publishing.
