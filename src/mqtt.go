package src

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type OnRequestHandler func([]byte)

type Mqtt struct {
	id  string
	url string

	client MQTT.Client

	onRequest OnRequestHandler
}

func (c *Mqtt) Init() {
	// options
	o := MQTT.NewClientOptions()

	uu, err := url.Parse(c.url)
	if err != nil {
		log.Fatalf("mqtt url parse error %s",err)
	}

	up, upe := uu.User.Password()

	// set server props
	var b string
	if uu.Scheme == "mqtts" {
		b = fmt.Sprintf("ssl://%s:%s", uu.Hostname(), uu.Port())
	} else {
		b = fmt.Sprintf("tcp://%s:%s", uu.Hostname(), uu.Port())
	}
	o.AddBroker(b)
	o.SetClientID(fmt.Sprintf("device-%s", c.id))
	o.SetUsername(uu.User.Username())
	if upe {
		o.SetPassword(up)
	}

	// set callback
	o.OnConnect = func(client MQTT.Client) {
		log.Println("mqtt connected ", c.id)
	}
	o.OnConnectionLost = func(client MQTT.Client, err error) {
		log.Println("mqtt connections lost: ", err)
	}

	// create client
	c.client = MQTT.NewClient(o)

	// connect
	token := c.client.Connect()
	token.Wait()
	if token.Error() != nil {
		log.Fatalf("connect error %s", token.Error())
	}

	// subscribe request
	c.subscribe("request", func(cc MQTT.Client, msg MQTT.Message) {
		if c.onRequest != nil {
			c.onRequest(msg.Payload())
		}
	})

	// send a heartbeat
	c.PublishHeartbeat()
}

func (c *Mqtt) Close() {
	// send offline
	c.PublishOffline()

	// disconnect
	c.client.Disconnect(250)
}

func (c *Mqtt) useTopic(prop string) string {
	return fmt.Sprintf("device/%s/%s", c.id, prop)
}

func (c *Mqtt) publish(prop string, message any) {
	j, err := json.Marshal(message)
	if err != nil {
		log.Fatalf("json string error %s", err)
	}

	token := c.client.Publish(c.useTopic(prop), 0, false, j)
	token.Wait()
	if token.Error() != nil {
		log.Printf("publish error %s", token.Error())
	}
}

func (c *Mqtt) subscribe(prop string, cb MQTT.MessageHandler) {
	token := c.client.Subscribe(c.useTopic(prop), 1, cb)
	token.Wait()
	if token.Error() != nil {
		log.Printf("subscribe error %s", token.Error())
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
