# 前端接口

这是前端接口文档

前端指的是边缘端设备上的触摸屏

前端程序负责在触摸屏上显示内容，并在用户操作时，发出消息

前端程序负责监视物理按钮，并在用户操作物理按钮时，发出消息

## 启动和停止

前端程序由主程序通过 cli 启动

主程序结束时会向前端进程发送信号`SIGINT`

## 交互

### 参数

启动前端程序时会输入以下参数

1. 主程序版本 `-mv <version>`
2. 消息路径 `-mp <path>`

### 消息

所有消息通过 unix socket 交互

所有数值为小端

消息分为 header 和 body 两部分, `header.body_size`指的是消息 body 的字节长度

#### 状态

1. 主程序定期发送`FRONT_MESSAGE_HEADER_TYPE_STATUS`，消息体为`front_message_status_t`

#### 语音输入

1. 用户按下按钮后，前端发送`FRONT_MESSAGE_HEADER_TYPE_TRANSCRIPT_START`
2. 用户点击停止，前端发送`FRONT_MESSAGE_HEADER_TYPE_TRANSCRIPT_STOP`
3. 用户点击取消，前端发送`FRONT_MESSAGE_HEADER_TYPE_TRANSCRIPT_CANCEL`
4. 语音输入传输数据，主程序发送`FRONT_MESSAGE_HEADER_TYPE_TRANSCRIPT_DATA`，消息体为字符串
5. 语音输入传输数据的 body_size 为 0，表示新段落
6. 语音输入结束后，主程序发送`FRONT_MESSAGE_HEADER_TYPE_TRANSCRIPT_END`

#### 事件日志

1. 主程序发送`FRONT_MESSAGE_HEADER_TYPE_LOG`，消息体为字符串

#### Agent

待定

#### 审批

1. 主程序发送`FRONT_MESSAGE_HEADER_TYPE_APPROVAL`，消息体为`front_message_approval_t`
2. 用户同意后，前端发送`FRONT_MESSAGE_HEADER_TYPE_APPROVAL_ACCEPT`
3. 用户拒绝后，前端发送`FRONT_MESSAGE_HEADER_TYPE_APPROVAL_DENY`
4. 用户取消后，前端发送`FRONT_MESSAGE_HEADER_TYPE_APPROVAL_CANCEL`
5. 审批结束后，主程序发送`FRONT_MESSAGE_HEADER_TYPE_APPROVAL_END`

#### 录制工作流

1. 用户开始录制工作流，前端发送`FRONT_MESSAGE_HEADER_TYPE_WORKFLOW_RECORD_START`
2. 用户保存工作流，前端发送`FRONT_MESSAGE_HEADER_TYPE_WORKFLOW_RECORD_SAVE`
3. 用户暂停录制工作流，前端发送`FRONT_MESSAGE_HEADER_TYPE_WORKFLOW_RECORD_PAUSE`

#### 错误

1. 错误时，发送`FRONT_MESSAGE_HEADER_TYPE_ERROR`，消息体为错误内容

## 参考实现

```h
// api.h
#include <stdint.h>
#include <string.h>
#include <time.h>

// 消息类型常量定义
#define FRONT_MESSAGE_HEADER_TYPE_STATUS 0x00000001

#define FRONT_MESSAGE_HEADER_TYPE_TRANSCRIPT_START 0x10000001
#define FRONT_MESSAGE_HEADER_TYPE_TRANSCRIPT_STOP 0x10000002
#define FRONT_MESSAGE_HEADER_TYPE_TRANSCRIPT_CANCEL 0x10000003
#define FRONT_MESSAGE_HEADER_TYPE_TRANSCRIPT_END 0x10000004
#define FRONT_MESSAGE_HEADER_TYPE_TRANSCRIPT_DATA 0x10000010

#define FRONT_MESSAGE_HEADER_TYPE_LOG 0x20000000

#define FRONT_MESSAGE_HEADER_TYPE_AGENT_LIST 0x30000000

#define FRONT_MESSAGE_HEADER_TYPE_APPROVAL 0x40000000
#define FRONT_MESSAGE_HEADER_TYPE_APPROVAL_ACCEPT 0x40000001
#define FRONT_MESSAGE_HEADER_TYPE_APPROVAL_DENY 0x40000002
#define FRONT_MESSAGE_HEADER_TYPE_APPROVAL_CANCEL 0x40000003
#define FRONT_MESSAGE_HEADER_TYPE_APPROVAL_END 0x40000004

#define FRONT_MESSAGE_HEADER_TYPE_WORKFLOW_RECORD_START 0x50000000
#define FRONT_MESSAGE_HEADER_TYPE_WORKFLOW_RECORD_SAVE 0x50000001
#define FRONT_MESSAGE_HEADER_TYPE_WORKFLOW_RECORD_PAUSE 0x50000002

#define FRONT_MESSAGE_HEADER_TYPE_ERROR 0xffffffff

// 状态常量定义
#define FRONT_MESSAGE_STATUS_SYSTEM_UNKNOWN 0x0
#define FRONT_MESSAGE_STATUS_SYSTEM_OFFLINE 0x1
#define FRONT_MESSAGE_STATUS_SYSTEM_ONLINE 0x2

#define FRONT_MESSAGE_STATUS_HDMI_UNKNOWN 0x0
#define FRONT_MESSAGE_STATUS_HDMI_NO_SIGNAL 0x1
#define FRONT_MESSAGE_STATUS_HDMI_CONNECTED 0x2

#define FRONT_MESSAGE_STATUS_USB_UNKNOWN 0x0
#define FRONT_MESSAGE_STATUS_USB_DISCONNECTED 0x1
#define FRONT_MESSAGE_STATUS_USB_CONNECTED 0x2

#define FRONT_MESSAGE_STATUS_WIFI_UNKNOWN 0x0
#define FRONT_MESSAGE_STATUS_WIFI_DISABLE 0x1
#define FRONT_MESSAGE_STATUS_WIFI_CONNECTING 0x2
#define FRONT_MESSAGE_STATUS_WIFI_CONNECTED 0x3

// 大小常量定义
#define FRONT_MESSAGE_HEADER_SIZE 4 + 8 + 4 + 4
#define FRONT_MESSAGE_STATUS_SIZE 4 + 4 + 4 + 4

// 消息头结构体
typedef struct
{
    uint32_t id;
    uint64_t timestamp;
    uint32_t msg_type;
    uint32_t body_size;
} front_message_header_t;

// 状态结构体
typedef struct
{
    uint32_t system;
    uint32_t hdmi;
    uint32_t usb;
    uint32_t wifi;
} front_message_status_t;

// 审批结构体
typedef struct
{
    uint32_t id;
    char *app;
    char *title;
    char *desc;
} front_message_approval_t;

/**
 * @brief 将消息头转换为字节数组
 *
 * @param header 消息头结构体指针
 * @param buffer 输出缓冲区，大小至少为FRONT_MESSAGE_HEADER_SIZE
 * @return 0成功，-1失败
 */
int front_message_header_to_bytes(const front_message_header_t *header, uint8_t *buffer);

/**
 * @brief 从字节数组解析消息头
 *
 * @param header 消息头结构体指针
 * @param buffer 输入缓冲区，大小至少为FRONT_MESSAGE_HEADER_SIZE
 * @return 0成功，-1失败
 */
int parse_front_message_header(front_message_header_t *header, const uint8_t *buffer);

/**
 * @brief 从字节数组解析状态结构体
 *
 * @param status 状态结构体指针
 * @param buffer 输入缓冲区，大小至少为FRONT_MESSAGE_STATUS_SIZE
 * @return 0成功，-1失败
 */
int parse_front_message_status(front_message_status_t *status, const uint8_t *buffer);

/**
 * @brief 从字节数组解析审批结构体
 *
 * @param approval 审批结构体指针
 * @param buffer 输入缓冲区
 * @return 0成功，-1失败
 */
int parse_front_message_approve(front_message_approval_t *approval, const uint8_t *buffer);

/**
 * @brief 释放审批结构体
 *
 * @param approval 结构体指针
 */
void destory_front_message_approve(front_message_approval_t *approval);
```

```c

// api.c
#include "api.h"

int front_message_header_to_bytes(const front_message_header_t *header, uint8_t *buffer)
{
    if (!header || !buffer)
    {
        return -1;
    }

    // 使用memcpy确保小端序存储
    memcpy(buffer, &header->id, sizeof(uint32_t));
    memcpy(buffer + 4, &header->timestamp, sizeof(uint64_t));
    memcpy(buffer + 12, &header->msg_type, sizeof(uint32_t));
    memcpy(buffer + 16, &header->body_size, sizeof(uint32_t));

    return 0;
}

int parse_front_message_header(front_message_header_t *header, const uint8_t *buffer)
{
    if (!header || !buffer)
    {
        return -1;
    }

    // 直接使用memcpy从字节数组复制到结构体成员
    memcpy(&header->id, buffer, sizeof(uint32_t));
    memcpy(&header->timestamp, buffer + 4, sizeof(uint64_t));
    memcpy(&header->msg_type, buffer + 12, sizeof(uint32_t));
    memcpy(&header->body_size, buffer + 16, sizeof(uint32_t));

    return 0;
}

int parse_front_message_status(front_message_status_t *status, const uint8_t *buffer)
{
    if (!status || !buffer)
    {
        return -1;
    }

    memcpy(&status->system, buffer, sizeof(uint32_t));
    memcpy(&status->hdmi, buffer + 4, sizeof(uint32_t));
    memcpy(&status->usb, buffer + 8, sizeof(uint32_t));
    memcpy(&status->wifi, buffer + 12, sizeof(uint32_t));

    return 0;
}

void destory_front_message_approve(front_message_approval_t *approval)
{
    if (!approval)
    {
        return;
    }

    if (approval->app)
    {
        free(approval->app);
    }
    if (approval->title)
    {
        free(approval->title);
    }
    if (approval->desc)
    {
        free(approval->desc);
    }

    free(approval);
}

int parse_front_message_approve(front_message_approval_t *approval, const uint8_t *buffer)
{
    if (!approval || !buffer)
    {
        return -1;
    }

    uint32_t offset = 0;

    // 读取id
    memcpy(&approval->id, buffer + offset, sizeof(uint32_t));
    offset += 4;

    // 读取app长度
    uint32_t app_len;
    memcpy(&app_len, buffer + offset, sizeof(uint32_t));
    offset += 4;

    // 读取app字符串
    if (app_len > 0)
    {
        approval->app = (char *)malloc(app_len + 1); // +1 for null terminator
        if (!approval->app)
        {
            return -1;
        }
        memcpy(approval->app, buffer + offset, app_len);
        approval->app[app_len] = '\0'; // 添加字符串结束符
        offset += app_len;
    }
    else
    {
        approval->app = (char *)malloc(1);
        if (approval->app)
        {
            approval->app[0] = '\0';
        }
    }

    // 读取title长度
    uint32_t title_len;
    memcpy(&title_len, buffer + offset, sizeof(uint32_t));
    offset += 4;

    // 读取title字符串
    if (title_len > 0)
    {
        approval->title = (char *)malloc(title_len + 1);
        if (!approval->title)
        {
            destory_front_message_approve(approval);
            return -1;
        }
        memcpy(approval->title, buffer + offset, title_len);
        approval->title[title_len] = '\0';
        offset += title_len;
    }
    else
    {
        approval->title = (char *)malloc(1);
        if (approval->title)
        {
            approval->title[0] = '\0';
        }
    }

    // 读取desc长度
    uint32_t desc_len;
    memcpy(&desc_len, buffer + offset, sizeof(uint32_t));
    offset += 4;

    // 读取desc字符串
    if (desc_len > 0)
    {
        approval->desc = (char *)malloc(desc_len + 1);
        if (!approval->desc)
        {
            destory_front_message_approve(approval);
            return -1;
        }
        memcpy(approval->desc, buffer + offset, desc_len);
        approval->desc[desc_len] = '\0';
        offset += desc_len;
    }
    else
    {
        approval->desc = (char *)malloc(1);
        if (approval->desc)
        {
            approval->desc[0] = '\0';
        }
    }

    return 0;
}
```
