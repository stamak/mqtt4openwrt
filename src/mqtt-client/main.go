package main

import (
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"io/ioutil"
	"log"
	"log/syslog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type SystemUsage struct {
	CPUUsage    string
	MemoryUsage string
	Download    string
	Upload      string
	WifiClients string
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

// Calculate CPU usage
var (
	cpuPrevTotal int64
	cpuPrevIdle  int64
)

func calculateCPUUsage() (int, error) {
	contents, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(contents), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == "cpu" {
			var total int64 = 0
			for i := 1; i < len(fields); i++ {
				value, err := strconv.ParseInt(fields[i], 10, 64)
				if err != nil {
					return 0, err
				}
				total += value
			}

			idle, err := strconv.ParseInt(fields[4], 10, 64)
			if err != nil {
				return 0, err
			}

			// Calculate CPU usage percentage
			cpuUsage := 100.0 * (float64(total-cpuPrevTotal) - float64(idle-cpuPrevIdle)) / float64(total-cpuPrevTotal)

			// Update previous values and measurement time
			cpuPrevTotal = int64(total)
			cpuPrevIdle = int64(idle)

			return int(cpuUsage), nil
		}
	}

	return 0, fmt.Errorf("unable to calculate CPU usage")
}

func getMemoryUsage() (int, error) {
	data, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	var available int = 0
	var total int = 0
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 1 && fields[0] == "MemTotal:" {
			total, err = strconv.Atoi(fields[1])
			if err != nil {
				return 0, err
			}
		}
		if len(fields) > 1 && fields[0] == "MemAvailable:" {
			available, err = strconv.Atoi(fields[1])
			if err != nil {
				return 0, err
			}
		}
	}
	if total == 0 {
		return 0, fmt.Errorf("Total memory is zero")
	}
	memoryUsage := 100.0 * (1.0 - float64(available)/float64(total))
	return int(memoryUsage), nil
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("Connect lost: %v", err)
}

func publish(client mqtt.Client, topic string, payload []byte) {
	token := client.Publish(topic, 0, false, payload)
	token.Wait()
	time.Sleep(time.Second)
}

func getSpeeds(ifaceName string) (float64, float64, error) {
	// Get initial byte counts for received and sent traffic
	startTime := time.Now()
	startRxBytes, err := getInterfaceBytes(ifaceName, "rx_bytes")
	if err != nil {
		log.Println("Error getting initial received byte count:", err)
		return 0, 0, nil
	}
	startTxBytes, err := getInterfaceBytes(ifaceName, "tx_bytes")
	if err != nil {
		log.Println("Error getting initial sent byte count:", err)
		return 0, 0, nil
	}

	// Wait for 1 second
	time.Sleep(time.Second)

	// Get final byte counts
	endRxBytes, err := getInterfaceBytes(ifaceName, "rx_bytes")
	if err != nil {
		log.Println("Error getting final received byte count:", err)
		return 0, 0, nil
	}
	endTxBytes, err := getInterfaceBytes(ifaceName, "tx_bytes")
	if err != nil {
		log.Println("Error getting final sent byte count:", err)
		return 0, 0, nil
	}

	// Calculate elapsed time in seconds
	elapsed := time.Since(startTime).Seconds()

	// Calculate bytes per second for download and upload
	downloadBps := float64(endRxBytes-startRxBytes) / elapsed
	uploadBps := float64(endTxBytes-startTxBytes) / elapsed

	// Convert to Mbps and format the output
	downloadMbps := downloadBps / 1024 / 1024 * 8 // Convert to megabits
	uploadMbps := uploadBps / 1024 / 1024 * 8     // Convert to megabits
	//fmt.Printf("Download speed on %s: %.2f Mbps\n", ifaceName, downloadMbps)
	//fmt.Printf("Upload speed on %s: %.2f Mbps\n", ifaceName, uploadMbps)
	return downloadMbps, uploadMbps, nil
}

// Please develop a separate function to get cpu usage and memory usage for linux

func getInterfaceBytes(ifaceName string, statName string) (int64, error) {
	path := fmt.Sprintf("/sys/class/net/%s/statistics/%s", ifaceName, statName)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}

	// Remove newline character before parsing to integer
	bytesStr := strings.TrimSuffix(string(data), "\n")

	bytes, err := strconv.ParseInt(bytesStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return bytes, nil
}

func getEnvVar(name string, defaultValues ...string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		if len(defaultValues) > 0 {
			return defaultValues[0], nil
		}
		return "", fmt.Errorf("environment variable %s is not set", name)
	}
	return value, nil
}

func getWifiClents() (int, error) {

	cmd := exec.Command("/etc/config/snmp_wifi_clients.sh")
	out, err := cmd.Output()
	if err != nil {
		log.Println("Error getting wifi clients:", err)
		return 0, nil
	}
	clients, err := strconv.Atoi(string(out))
	if err != nil {
		log.Println("Error converting wifi clients:", err)
		return 0, nil
	}
	return clients, nil
}

func main() {
	// Logging
	syslogger, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "mqtt-metric-agent")
	if err != nil {
		log.Fatalln(err)
	}
	log.SetOutput(syslogger)
	log.SetFlags(0)

	// Configuration
	broker, err := getEnvVar("MQTT_BROKER", "127.0.0.1")
	if err != nil {
		log.Fatalln(err)
	}
	portStr, err := getEnvVar("MQTT_PORT", "1883")
	if err != nil {
		log.Fatalln(err)
	}
	port, _ := strconv.Atoi(portStr)
	username, err := getEnvVar("MQTT_USER", "homeassistant")
	if err != nil {
		log.Fatalln(err)
	}
	password, err := getEnvVar("MQTT_PASS")
	if err != nil {
		log.Fatalln(err)
	}
	topic, err := getEnvVar("MQTT_TOPIC", "routers/some_router/some_topic")
	if err != nil {
		log.Fatalln(err)
	}
	//
	ifaceName, err := getEnvVar("IFACE_NAME", "wan")
	if err != nil {
		log.Fatalln(err)
	}
	sleep_time := 5 // wait time
	sleep_time_str, err := getEnvVar("SLEEP_TIME", "5")
	if err != nil {
		log.Fatalln(err)
	}
	sleep_time, _ = strconv.Atoi(sleep_time_str)
	log.Printf("MQTT Broker: %s, Port: %d,"+
		"Username: %s, Password: ***,"+
		"Topic: %s, sleep_time: %d", broker, port, username, topic, sleep_time)

	log.Println("Starting MQTT service ...")
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	opts.SetClientID("go_mqtt_client-" + strconv.FormatInt(time.Now().Unix(), 10))
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	for {
		select {
		//case <-time.Tick(5 * time.Second): // Produce high load average
		case <-time.After(time.Duration(sleep_time) * time.Second):
			// Calculate download and upload speeds
			downloadMbps, uploadMbps, err := getSpeeds(ifaceName)
			if err != nil {
				log.Println("Error getting speed:", err)
			}
			cpuUsage, err := calculateCPUUsage()
			if err != nil {
				log.Println("Error getting CPU usage:", err)
			}
			memoryUsage, err := getMemoryUsage()
			if err != nil {
				log.Println("Error getting memory usage:", err)
			}
			// wifiClients, err := getWifiClents()
			// if err != nil {
			// 	log.Println("Error getting wifi clients:", err)
			// }
			// Do not use wifi clients for now as it takes a lot of time to get the value
			wifiClients := 0

			body := &SystemUsage{Download: fmt.Sprintf("%.2f", downloadMbps), Upload: fmt.Sprintf("%.2f", uploadMbps),
				CPUUsage: fmt.Sprintf("%d", cpuUsage), MemoryUsage: fmt.Sprintf("%d", memoryUsage),
				WifiClients: fmt.Sprintf("%d", wifiClients)}
			payload, err := json.Marshal(body)
			if err != nil {
				log.Println(err)
				return
			}

			publish(client, topic, payload)

		}
	}
	client.Disconnect(250)
}
