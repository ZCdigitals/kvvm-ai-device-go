# KVVM.AI Device Go

## Develop

### Dependences

#### VPx

```bash
sudo apt install libvpx-dev
```

#### X11

```bash
sudo apt install libx11-dev libxext-dev
```

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
```

## Build

### Tool chain

```bash
sudo apt search gcc-arm-linux-gnueabihf g++-arm-linux-gnueabihf
```
