package src

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type OnRequestHandler func([]byte)

type Mqtt struct {
	ssl      bool
	broker   string
	port     int64
	deviceId string
	username string
	password string

	client MQTT.Client

	OnRequest OnRequestHandler
}

func (c *Mqtt) Init() {
	// options
	options := MQTT.NewClientOptions()

	// set server props
	var b string
	if c.ssl {
		b = fmt.Sprintf("ssl://%s:%d", c.broker, c.port)
	} else {
		b = fmt.Sprintf("tcp://%s:%d", c.broker, c.port)
	}
	options.AddBroker(b)
	options.SetClientID(fmt.Sprintf("device-%s", c.deviceId))
	options.SetUsername(c.username)
	options.SetPassword(c.password)

	// set callback
	options.OnConnect = c.onConnect
	options.OnConnectionLost = c.onConnectionLost

	// create client
	c.client = MQTT.NewClient(options)

	// connect
	token := c.client.Connect()
	token.Wait()
	if token.Error() != nil {
		log.Fatal("connect error ", token.Error())
	}

	// subscribe request
	c.subscribe("request", c.onRequest)
	// send heartbeat
	c.PublishHeartbeat()

}

func (c *Mqtt) Close() {
	// send offline
	c.PublishOffline()

	// disconnect
	c.client.Disconnect(250)
}

func (c *Mqtt) useTopic(prop string) string {
	return fmt.Sprintf("device/%s/%s", c.deviceId, prop)
}

func (c *Mqtt) publish(prop string, message any) {
	j, err := json.Marshal(message)
	if err != nil {
		log.Fatal("json string error ", err)
	}

	token := c.client.Publish(c.useTopic(prop), 0, false, j)
	token.Wait()
	if token.Error() != nil {
		log.Fatal("publish error ", token.Error())
	}
}

func (c *Mqtt) subscribe(prop string, cb MQTT.MessageHandler) {
	token := c.client.Subscribe(c.useTopic(prop), 1, cb)
	token.Wait()
	if token.Error() != nil {
		log.Fatal("subscribe error ", token.Error())
	}
}

type MqttStatus struct {
	Time   int64 `json:"time"`
	Status bool  `json:"status"`
}

func (c *Mqtt) PublishOnline() {
	s := MqttStatus{Status: true, Time: time.Now().Unix()}
	c.publish("status", s)
}

func (c *Mqtt) PublishOffline() {
	s := MqttStatus{Status: false, Time: time.Now().Unix()}
	c.publish("status", s)
}

type MqttHeartbeat struct {
	Time int64 `json:"time"`
}

func (c *Mqtt) PublishHeartbeat() {
	s := MqttHeartbeat{Time: time.Now().Unix()}
	c.publish("heartbeat", s)
}

func (c *Mqtt) PublishResponse(data any) {
	c.publish("response", data)
}

func (c *Mqtt) onConnect(client MQTT.Client) {
	log.Println("mqtt connected ", c.deviceId)
}

func (c *Mqtt) onConnectionLost(client MQTT.Client, err error) {
	log.Println("mqtt connections lost: ", err)
}

func (c *Mqtt) onRequest(client MQTT.Client, message MQTT.Message) {
	if c.OnRequest != nil {
		c.OnRequest(message.Payload())
	}
}
