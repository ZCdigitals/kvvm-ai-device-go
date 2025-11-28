// 包声明，可执行程序必须是 package main
package main

import (
	"device-go/src"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	args := src.ParseArgs()

	if args.Version {
		log.Println(src.VersionLong())
		return
	}

	// 初始化
	d := src.Device{Id: args.Id, MqttUrl: "mqtt", WsUrl: "ws", WsKey: "wsKey", MediaSource: src.DeviceMediaSource(args.MediaSource)}
	d.Init()

	// 退出
	defer d.Close()

	// 等待中断信号
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}
