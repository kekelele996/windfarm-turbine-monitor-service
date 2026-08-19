# Bug 复现说明

## Bug 是什么

告警组件 map 未初始化。

## 如何触发

空分发器/去重器/路由器/通知器首次写入。

## 错误信息

panic: assignment to entry in nil map。
