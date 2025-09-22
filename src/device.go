package src

import (
	"log"
	"net/url"
	"strconv"
)

type Device struct {
	Id         string
	MqttBroker string

	mqtt Mqtt
}

func (d *Device) Init() {
	uu, err := url.Parse(d.MqttBroker)
	if err != nil {
		log.Fatal(err)
	}

	p, err := strconv.ParseInt(uu.Port(), 0, 64)
	if err != nil {
		log.Fatal(err)
	}
	up, _ := uu.User.Password()

	d.mqtt = Mqtt{
		broker:    uu.Hostname(),
		port:      p,
		deviceId:  d.Id,
		username:  uu.User.Username(),
		password:  up,
		OnRequest: d.onRequest,
	}

	d.mqtt.Init()
}

func (d *Device) Close() {
	d.mqtt.Close()
}

func (d *Device) onRequest(m *Mqtt, r *MqttRequest) {
	log.Println("request", r)
}
