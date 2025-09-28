#!/bin/bash

echo "=== 配置 USB HID  ==="

# 定义路径变量
GADGET_PATH="/sys/kernel/config/usb_gadget/rockchip"
CONFIG_PATH="$GADGET_PATH/configs/b.1"
DESCRIPTOR_PATH="usb.bin"

# 1. 检查必要目录是否存在
if [ ! -d "$GADGET_PATH" ]; then
    echo "错误: Rockchip gadget 路径不存在: $GADGET_PATH"
    exit 1
fi

# 2. 创建 HID 功能
echo "创建 HID 功能..."
mkdir -p $GADGET_PATH/functions/hid.usb1

# 3. 配置参数
# echo "0x2207" > $GADGET_PATH/idVendor   # embed fire
# echo "0x0200" > $GADGET_PATH/bcdUSB   # USB 2.0
# echo "0x0310" > $GADGET_PATH/bcdDevice   # 3.1
echo "0xEF" > $GADGET_PATH/bDeviceClass   # 其他设备
echo "0x02" > $GADGET_PATH/bDeviceSubClass  # 通用设备
echo "0X01" > $GADGET_PATH/bDeviceProtocol  # 接口代理描述
echo 0 > $GADGET_PATH/functions/hid.usb1/subclass    # Boot 接口
echo 0 > $GADGET_PATH/functions/hid.usb1/protocol    # 协议
echo 8 > $GADGET_PATH/functions/hid.usb1/report_length # 报告长度

# 4. 设置报告描述符
echo "设置报告描述符..."
cat $DESCRIPTOR_PATH > $GADGET_PATH/functions/hid.usb1/report_desc

# 5. 将功能链接到配置
echo "链接功能到配置..."
ln -s $GADGET_PATH/functions/hid.usb1 $CONFIG_PATH/

# 6. 启用/重新启用 gadget
echo "启用 USB gadget..."
UDC_CONTROLLER=$(ls /sys/class/udc/ | head -n1)
if [ -n "$UDC_CONTROLLER" ]; then
    # 先禁用（如果已启用）
    echo "" > $GADGET_PATH/UDC 2>/dev/null
    sleep 1
    # 重新启用
    echo $UDC_CONTROLLER > $GADGET_PATH/UDC
    echo "使用 UDC 控制器: $UDC_CONTROLLER"
else
    echo "错误: 未找到可用的 UDC 控制器"
    exit 1
fi

# 7. 等待设备节点创建
echo "等待设备节点创建..."
sleep 2

# 8. 验证配置
echo "验证配置:"
if [ -c /dev/hidg0 ]; then
    echo "✓ HID 设备 0 已创建: /dev/hidg0"
    chmod 666 /dev/hidg0
    echo "配置成功完成!"
else
    echo "⚠️  HID 设备未创建，但配置可能仍部分成功"
    echo "检查内核消息: dmesg | tail -10"
fi
