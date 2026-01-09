#!/bin/bash

/root/device \
	--serve-url https://kvvm-ai-serve.zcdigitals.com \
	--serve-client-id 6952402ffd6177d4d287b95e \
	--mqtt-url mqtts://device:device12345@kvvm-ai-base.zcdigitals.com:8883 \
	--config-path /root/kvvm-ai.json \
	--hid-path /dev/hidg1
