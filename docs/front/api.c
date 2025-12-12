
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
