package src

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type MqttOnRequest func(payload []byte)

type Mqtt struct {
	id  string
	url string

	client MQTT.Client

	onRequest MqttOnRequest
}

func (c *Mqtt) openClient() error {
	// options
	o := MQTT.NewClientOptions()

	uu, err := url.Parse(c.url)
	if err != nil {
		log.Println("mqtt url parse error", err)
		return err
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
		log.Println("mqtt connected")
	}
	o.OnConnectionLost = func(client MQTT.Client, err error) {
		log.Println("mqtt connections lost", err)
	}

	// create client
	c.client = MQTT.NewClient(o)

	// connect
	token := c.client.Connect()
	token.Wait()
	err = token.Error()
	if err != nil {
		log.Println("mqtt connect error", err)
		return err
	}

	return nil
}

func (c *Mqtt) useTopic(prop string) string {
	return fmt.Sprintf("device/%s/%s", c.id, prop)
}

func (c *Mqtt) publish(prop string, message any) error {
	j, err := json.Marshal(message)
	if err != nil {
		log.Println("mqtt json marshal error", err)
		return err
	}

	token := c.client.Publish(c.useTopic(prop), 0, false, j)
	token.Wait()
	err = token.Error()
	if err != nil {
		log.Println("mqtt publish error", err)
		return err
	}

	return nil
}

func (c *Mqtt) subscribe(prop string, cb MQTT.MessageHandler) error {
	token := c.client.Subscribe(c.useTopic(prop), 1, cb)
	token.Wait()
	err := token.Error()
	if err != nil {
		log.Println("mqtt subscribe error", err)
		return err
	}

	return nil
}

type mqttStatus struct {
	Time   int64 `json:"time"`
	Status bool  `json:"status"`
}

// there is no neet to publish online, use `publishHeartbeat`
// func (c *Mqtt) publishOnline() {
// 	s := mqttStatus{Status: true, Time: time.Now().Unix()}
// 	c.publish("status", s)
// }

func (c *Mqtt) publishOffline() error {
	s := mqttStatus{Status: false, Time: time.Now().Unix()}
	return c.publish("status", s)
}

type mqttHeartbeat struct {
	Time int64 `json:"time"`
}

func (c *Mqtt) publishHeartbeat() error {
	s := mqttHeartbeat{Time: time.Now().Unix()}
	return c.publish("heartbeat", s)
}

func (c *Mqtt) subscribeRequest() error {
	return c.subscribe("request", func(cc MQTT.Client, msg MQTT.Message) {
		if c.onRequest != nil {
			c.onRequest(msg.Payload())
		}
	})
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
		return err
	}
	err = c.publishHeartbeat()

	return err
}

func (c *Mqtt) Close() {
	// send offline
	c.publishOffline()

	// disconnect
	c.client.Disconnect(250)
}

func (c *Mqtt) Send(data any) error {
	return c.publishResponse(data)
}
