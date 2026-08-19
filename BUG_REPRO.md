# Bug 复现说明

## Bug 是什么

阈值 map 未初始化。

## 如何触发

用只填部分字段的 JSON 配置加载后写入阈值。

## 错误信息

panic: assignment to entry in nil map。
