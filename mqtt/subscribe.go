package mqtt

import (
	"context"
	"crypto/tls"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/domgoodwin/go-automation/influx"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

const (
	server   = "tcp://192.168.0.240:1883"
	qos      = 0
	password = "Sighing-Mulled-Arguable8"
)

func onMessageReceived(client MQTT.Client, message MQTT.Message) {

	dataType := ""

	switch message.Topic() {
	case "solar_assistant/inverter_1/pv_power/state",
		"solar_assistant/inverter_1/pv_power_1/state",
		"solar_assistant/inverter_1/pv_power_2/state":
		dataType = "pv"
	case "solar_assistant/inverter_1/grid_power/state":
		dataType = "grid"
	case "solar_assistant/inverter_1/load_power/state",
		"solar_assistant/inverter_1/load_percentage/state":
		dataType = "load"
	case "solar_assistant/total/battery_state_of_charge/state",
		"solar_assistant/total/battery_power/state":
		dataType = "battery"
	default:
		log.Debug("Dropping message on topic: %s\nMessage: %s\n", message.Topic(), message.Payload())
		return
	}

	value, err := strconv.ParseInt(string(message.Payload()), 10, 64)
	if err != nil {
		log.Error("error parsing value %v as int", message.Payload())
	}

	influx.Write(context.Background(), message.Topic(), map[string]string{"type": dataType}, map[string]interface{}{"state": value})
}

func Subscribe(topic string) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	hostname, _ := os.Hostname()
	clientID := hostname + strconv.Itoa(time.Now().Second())

	connOpts := MQTT.NewClientOptions().AddBroker(server).SetClientID(clientID).SetCleanSession(true)

	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	connOpts.SetTLSConfig(tlsConfig)

	connOpts.OnConnect = func(c MQTT.Client) {
		if token := c.Subscribe(topic, byte(qos), onMessageReceived); token.Wait() && token.Error() != nil {
			log.Errorf("error subscribing %v", token.Error())
			return
		}
	}

	client := MQTT.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	} else {
		log.Infof("Connected to %s\n", server)
	}

	<-c
	return nil
}
