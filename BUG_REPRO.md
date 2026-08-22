# 缺陷复现说明

## Bug 是什么

学生退出课程后再次报名，同一课程和学生的关系没有被恢复，新的写入会与保留记录冲突，导致再次报名失败。

## 如何触发

创建一名学生和一门课程，完成首次报名后退出课程，再以相同学生身份重新报名。

## 根因

报名生命周期同时保留历史关系和唯一约束，但重新报名路径没有恢复已有的非活动关系，而是继续创建新关系。服务编排与仓储查询的语义不一致，使历史记录既不能恢复也阻塞新的写入。

## 运行指令

```text
go test ./internal/service -v -run '^TestReenrollReactivatesOneRetainedEnrollment$' -count=1
```

## 错误信息

第二次报名触发唯一约束，测试报告重新报名失败。

## 错误堆栈

```text
internal/repository/enrollment_repository.go:19 UNIQUE constraint failed: enrollments.course_id, enrollments.student_id
enrollment_reentry_contract_test.go:53: re-enrollment failed: UNIQUE constraint failed: enrollments.course_id, enrollments.student_id
--- FAIL: TestReenrollReactivatesOneRetainedEnrollment
```
