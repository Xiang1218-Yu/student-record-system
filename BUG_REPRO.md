# Bug 复现说明

## Bug 是什么

删除仍负责 scheduled 课程的教师时，状态检查没有触发删除保护，教师被软删除后课程失去可用负责人；ongoing 课程的保护行为与 scheduled 不一致。

## 如何触发

在含 Bug 的环境中执行定向测试。测试通过教师删除入口、教师服务和课程状态查询构造 scheduled 与 ongoing 两种课程。

## 运行指令

```bash
go test -v ./internal/handler -run '^TestTeacherDeleteProtectsScheduledAndOngoingCourses$' -count=1
```

## 错误堆栈

以下错误信息来自上述运行指令在含 Bug 环境中的原始失败输出。

```text
=== RUN   TestTeacherDeleteProtectsScheduledAndOngoingCourses
    teacher_delete_contract_test.go:51: teacher teacher-007-scheduled deletion returned success while owning an active course
--- FAIL: TestTeacherDeleteProtectsScheduledAndOngoingCourses
FAIL
FAIL	course-attendance/internal/handler
```

## 根因

internal/models/course.go 的课程状态模型、internal/repository/course_repository.go 的教师活动课程统计、internal/service/teacher_service.go 的删除决策，以及 internal/handler/teacher_handler.go 的删除响应共同决定删除保护。仓储与服务只覆盖部分活动状态，scheduled 课程没有进入阻断条件，服务继续软删除教师，处理器又返回成功，最终课程仍存在但失去负责人，形成状态污染；同时冲突条件未被正确返回，构成错误传播缺失。

生产符号：Course 状态模型、CourseRepository 的教师活动课程统计、TeacherService 的删除决策和 TeacherHandler 的删除响应共同定义删除保护。
