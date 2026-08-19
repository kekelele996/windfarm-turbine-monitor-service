# Bug 复现说明

## Bug 是什么

摄取并发协调缺陷。

## 如何触发

并发调用摄取，或让 sink 失败，或停止周期刷盘。

## 错误信息

采样批量丢失、sink 错误被吞、停止前剩余采样未落盘（并发验证失败，部分场景 panic: negative WaitGroup counter）。
