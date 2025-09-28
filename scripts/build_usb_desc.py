import re
from typing import Optional


def build_hid_descriptor(input_path: str, output_path: Optional[str]):
    try:
        with open(input_path) as f:
            data = f.read()

        # 移除单行注释（//及其后面的所有内容）
        no_comments = re.sub(r"//.*", "", data)

        # 移除多行注释（/* ... */） - 虽然您的示例中没有，但为了通用性加上
        no_comments = re.sub(r"/\*.*?\*/", "", no_comments, flags=re.DOTALL)

        # 移除所有空白字符（空格、制表符、换行等）
        no_whitespace = re.sub(r"\s+", "", no_comments)

        # 提取所有十六进制字节（格式为0xXX或XX,）
        hex_bytes = re.findall(
            r"(0x[0-9A-Fa-f]{1,2}|[0-9A-Fa-f]{1,2},?)", no_whitespace
        )

        # 将十六进制字符串转换为整数
        byte_values = []
        for byte_str in hex_bytes:
            # 处理带0x前缀的格式
            if byte_str.startswith("0x"):
                byte_str = byte_str[2:]
            # 处理可能存在的逗号
            if byte_str.endswith(","):
                byte_str = byte_str[:-1]

            if byte_str:  # 确保不是空字符串
                byte_values.append(int(byte_str, 16))

        bs = bytearray(byte_values)

        if output_path:
            with open(output_path, "wb") as bf:
                bf.write(bs)
            print(f"成功将描述符保存到: {output_path}, 大小 {len(bs)}")
        else:
            print(f"描述符为")
            for i, b in enumerate(bs):
                if i % 16 == 0:
                    print(f"\n{i:04X}: ", end="")
                print(f"{b:02X} ", end="")
            print()

    except FileNotFoundError:
        print(f"错误: 找不到文件 '{input_path}'")
        return False
    except Exception as e:
        print(f"处理文件时发生错误: {str(e)}")
        return False


if __name__ == "__main__":
    build_hid_descriptor("config/usb.txt", "config/usb.bin")
