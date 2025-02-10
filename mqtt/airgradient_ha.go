package mqtt

import (
	MQTT "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

const (
	haTopicPrefix = "homeassistant"
)

var hasSentDiscovery = false

func processAirGradientMessage(client MQTT.Client, message MQTT.Message) error {
	log.Debugf("Processing air gradient message: %s\nMessage: %s\n", message.Topic(), message.Payload())
	return nil
	// msg := &AirGradientMessage{}

	// err := json.Unmarshal(message.Payload(), msg)
	// if err != nil {
	// 	return err
	// }
	// sendDiscoveryMessage(client, msg)

}

// func sendDiscoveryMessage(client MQTT.Client, msg *AirGradientMessage) {
// 	discoveryPath := fmt.Sprintf("%v/config", msg.TopicPrefix())
// 	payload := &HADiscovery{
// 		UniqID:     msg.UniqueID() + "_temperature",
// 		StatT:      fmt.Sprintf("%v/state"),
// 		DevCla:     "temperature",
// 		UnitOfMeas: "°C",
// 		Dev: HADiscoveryDevice{
// 			Mf:   "AirGradient",
// 			Mdl:  msg.Model,
// 			Name: "AirGradient ONE",
// 			Ids:  []string{msg.Serialno},
// 		},
// 	}
// 	token := client.Publish(discoveryPath, 1, true, payload)
// 	success := token.WaitTimeout(time.Second * 30)
// 	if !success {
// 		log.Errorf("Failed sending discovery message: %v", payload)
// 	}
// }

// func sendStateMessage(client MQTT.Client, msg *AirGradientMessage) {
// 	discoveryPath := fmt.Sprintf("%v/config", msg.TopicPrefix())
// 	payload := &HADiscovery{
// 		UniqueID:          msg.UniqueID() + "_temperature",
// 		StateTopic:        fmt.Sprintf("airgradient/readings/%v", msg.UniqueID()),
// 		DeviceClass:       "temperature",
// 		UnitOfMeasurement: "°C",
// 		ValueTemplate: "",
// 		Device: HADiscoveryDevice{
// 			Manufacturer: "AirGradient",
// 			Model:        msg.Model,
// 			Name:         "AirGradient ONE",
// 			Ids:          []string{msg.Serialno},
// 		},
// 	}
// 	token := client.Publish(discoveryPath, 1, true, payload)
// 	success := token.WaitTimeout(time.Second * 30)
// 	if !success {
// 		log.Errorf("Failed sending discovery message: %v", payload)
// 	}
// }

// func (a *AirGradientMessage) UniqueID() string {
// 	return fmt.Sprintf("air_gradient.%v.%v", a.Model, a.Serialno)
// }

// func (a *AirGradientMessage) TopicPrefix() string {
// 	return fmt.Sprintf("%v/sensor/%v", haTopicPrefix, a.UniqueID())
// }

// type AirGradientMessage struct {
// 	Wifi            int     `json:"wifi"`
// 	Serialno        string  `json:"serialno"`
// 	Rco2            int     `json:"rco2"`
// 	Pm01            int     `json:"pm01"`
// 	Pm02            int     `json:"pm02"`
// 	Pm10            int     `json:"pm10"`
// 	Pm003Count      int     `json:"pm003Count"`
// 	Atmp            float64 `json:"atmp"`
// 	AtmpCompensated float64 `json:"atmpCompensated"`
// 	Rhum            int     `json:"rhum"`
// 	RhumCompensated int     `json:"rhumCompensated"`
// 	TvocIndex       int     `json:"tvocIndex"`
// 	TvocRaw         int     `json:"tvocRaw"`
// 	NoxIndex        int     `json:"noxIndex"`
// 	NoxRaw          int     `json:"noxRaw"`
// 	Boot            int     `json:"boot"`
// 	BootCount       int     `json:"bootCount"`
// 	LedMode         string  `json:"ledMode"`
// 	Firmware        string  `json:"firmware"`
// 	Model           string  `json:"model"`
// }

// type HADiscovery struct {
// 	UniqueID          string            `json:"unique_id"`
// 	Name              string            `json:"name"`
// 	StateTopic        string            `json:"state_topic"`
// 	DeviceClass       string            `json:"device_class"`
// 	UnitOfMeasurement string            `json:"unit_of_measurement"`
// 	Device            HADiscoveryDevice `json:"device"`
// }

// type HADiscoveryDevice struct {
// 	Manufacturer string   `json:"manufacturer"`
// 	Model        string   `json:"model"`
// 	Name         string   `json:"name"`
// 	Ids          []string `json:"ids"`
// }
