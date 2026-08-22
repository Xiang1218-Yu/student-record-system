# 缺陷复现说明

## Bug 是什么

教师仍拥有进行中课程时，删除请求被当作成功处理，教师和课程的归属关系因此被破坏。

## 如何触发

创建带有进行中课程的教师，再调用该教师的删除入口。

## 根因

课程依赖检查遗漏进行中状态，服务层没有以冲突语义阻止删除，处理器最终返回成功响应。课程状态判断、删除决策和 HTTP 错误映射没有保持一致。

## 运行指令

```text
go test ./internal/handler -v -run '^TestTeacherDeleteBlocksOngoingCourseAndPreservesOwner$' -count=1
```

## 错误信息

删除拥有进行中课程的教师时返回成功，而不是冲突状态。

## 错误堆栈

```text
teacher_delete_contract_test.go:41: ongoing-course delete returned 200, want 409
--- FAIL: TestTeacherDeleteBlocksOngoingCourseAndPreservesOwner
```
