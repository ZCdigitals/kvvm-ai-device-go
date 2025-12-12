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
