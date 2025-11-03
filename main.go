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
	var url string
	var version bool

	flag.StringVar(&id, "device-id", "", "设备ID")
	flag.StringVar(&url, "mqtt-broker", "mqtt://device:device12345@localhost:1883", "MQTT代理地址")
	flag.BoolVar(&version, "version", false, "Print version")
	flag.Parse()

	if version {
		log.Println(src.VersionLong())
		return
	}

	// 初始化
	d := src.Device{Id: id, MqttUrl: url}
	d.Init()

	// 退出
	defer d.Close()

	// 等待中断信号
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}
