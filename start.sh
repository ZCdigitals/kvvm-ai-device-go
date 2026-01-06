#!/bin/bash

/root/device \
	--serve-url http://192.168.3.45:3000 \
	--serve-client-id 695229de6d72a035dc9d1b71 \
	--mqtt-url mqtt://device:device12345@192.168.3.45:1883 \
	--config-path /root/kvvm-ai.json \
	--hid-path /dev/hidg1
