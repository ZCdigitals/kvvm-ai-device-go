# KVVM.AI Device Go

## Build

if you need run on other platform, edit `build.sh`

## Deploy

copy the binary file `output/device` and `video` to target `/root/`

we use absolute path currently

or you may edit the `video` path in `media.go`

## Develop

### Dependences

#### Video

the `video` is built by `https://github.com/ZCdigitals/kvvm-ai-video`

### Certs

when deploied to public network

all communication must use tls when throgh public network, that is why the device must install certs.

or we will get

```
Error : tls: failed to verify certificate: x509: certificate signed by unknown authority
```

a easy way is copying from your own computer. copy `/usr/share/ca-certificates` and `/etc/ssl/certs/*`.

### USB

```bash
./script/setup_usb.sh
```

### V4l2

```bash
v4l2-ctl -d /dev/video0 --all
v4l2-ctl -d /dev/video0 --list-formats-ext
v4l2-ctl -d /dev/video0 --set-fmt-video=width=1920,height=1080,pixelformat=NV12
```

### Gstreamer

```bash
gst-launch-1.0 -v v4l2src device=/dev/video0 io-mode=mmap ! video/x-raw,format=NV12,width=1920,height=1080 ! mpph264enc gop=2 ! rtph264pay config-interval=-1 aggregate-mode=zero-latency ! udpsink host=127.0.0.1 port=5004

gst-inspect-1.0
```
