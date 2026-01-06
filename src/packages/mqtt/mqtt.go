package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	URL "net/url"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MqttOnRequest func(payload []byte)

type Mqtt struct {
	id string

	client mqtt.Client

	OnRequest MqttOnRequest
}

func NewMqtt(id string, url string) Mqtt {
	// options
	o := mqtt.NewClientOptions()

	uu, err := URL.Parse(url)
	if err != nil {
		log.Fatalln("mqtt url parse error", err)
	}

	up, upe := uu.User.Password()

	// set props
	switch uu.Scheme {
	case "mqtts":
		o.AddBroker(fmt.Sprintf("ssl://%s:%s", uu.Hostname(), uu.Port()))
		break
	case "mqtt":
		o.AddBroker(fmt.Sprintf("tcp://%s:%s", uu.Hostname(), uu.Port()))
		break
	default:
		log.Fatalln("mqtt unknown url schema %s", uu.Scheme)
	}
	o.SetClientID(id)
	o.SetUsername(uu.User.Username())
	if upe {
		o.SetPassword(up)
	}

	// set callback
	o.OnConnect = func(client mqtt.Client) {
		log.Println("mqtt connected")
	}
	o.OnConnectionLost = func(client mqtt.Client, err error) {
		log.Println("mqtt connections lost", err)
	}

	return Mqtt{
		id:     id,
		client: mqtt.NewClient(o),
	}
}

func (c *Mqtt) openClient() error {
	token := c.client.Connect()
	token.Wait()
	return token.Error()
}

func (c *Mqtt) closeClient() {
	c.client.Disconnect(250)
}

func (c *Mqtt) useTopic(prop string) string {
	return fmt.Sprintf("device/%s/%s", c.id, prop)
}

func (c *Mqtt) publish(prop string, message any) error {
	log.Println("mqtt publish", prop, message)

	j, err := json.Marshal(message)
	if err != nil {
		return err
	}

	token := c.client.Publish(c.useTopic(prop), 0, false, j)
	token.Wait()
	return token.Error()
}

func (c *Mqtt) subscribe(prop string, cb mqtt.MessageHandler) error {
	token := c.client.Subscribe(c.useTopic(prop), 1, cb)
	token.Wait()
	return token.Error()
}

type MqttMessage struct {
	Time int64 `json:"time"`
}

type MqttMessageStatus struct {
	Time   int64 `json:"time"`
	Status bool  `json:"status"`
}

// there is no neet to publish online, use `publishHeartbeat`
// func (c *Mqtt) publishOnline() {
// 	s := mqttStatus{Status: true, Time: time.Now().Unix()}
// 	c.publish("status", s)
// }

func (c *Mqtt) publishOffline() error {
	s := MqttMessageStatus{
		Status: false,
		Time:   time.Now().Unix(),
	}
	return c.publish("status", s)
}

func (c *Mqtt) publishHeartbeat() error {
	s := MqttMessage{Time: time.Now().Unix()}
	return c.publish("heartbeat", s)
}

func (c *Mqtt) subscribeRequest() error {
	return c.subscribe(
		"request",
		func(cc mqtt.Client, msg mqtt.Message) {
			if c.OnRequest != nil {
				c.OnRequest(msg.Payload())
			}
		},
	)
}

func (c *Mqtt) publishResponse(data any) error {
	return c.publish("response", data)
}

func (c *Mqtt) Open() error {
	err := c.openClient()
	if err != nil {
		return err
	}

	err = c.subscribeRequest()
	if err != nil {
		c.closeClient()
		return err
	}

	err = c.publishHeartbeat()
	if err != nil {
		c.closeClient()
		return err
	}

	return nil
}

func (c *Mqtt) Close() {
	// send offline
	c.publishOffline()

	c.closeClient()
}

func (c *Mqtt) Send(data any) error {
	return c.publishResponse(data)
}

func (c *Mqtt) IsConnected() bool {
	return c.client.IsConnected()
}
