// 包声明，可执行程序必须是 package main
package main

import (
	"device-go/src"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var id string
	var mqtt string
	var ws string
	var wsKey string
	var version bool

	flag.StringVar(&id, "device-id", "", "Device ID")
	flag.StringVar(&mqtt, "mqtt-broker", "", "MQTT broker url")
	// flag.StringVar(&mqtt, "mqtt-broker", "mqtt://device:device12345@localhost:1883", "MQTT broker url")
	flag.StringVar(&ws, "websocket", "ws://localhost:1883", "Websocket server url")
	flag.StringVar(&wsKey, "websocket-key", "", "Websocket key")
	flag.BoolVar(&version, "version", false, "Print version")
	flag.Parse()

	if version {
		log.Println(src.VersionLong())
		return
	}

	if (ws != "" && wsKey == "") || (ws == "" && wsKey != "") {
		log.Fatalln("Must input websocket-key when enable websocket")
	}

	// 初始化
	d := src.Device{Id: id, MqttUrl: mqtt, WsUrl: ws, WsKey: wsKey}
	d.Init()

	// 退出
	defer d.Close()

	// 等待中断信号
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}
