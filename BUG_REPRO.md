# Bug 复现说明

## Bug 是什么

扫码和教师补签并发到达时，同一课程中的同一学生可能生成两条出勤记录，课程统计因此重复计算。

## 如何触发

受保护的并发验证会通过真实签到服务和出勤仓储同时提交扫码与补签请求，并检查最终记录数和成功次数。

## 运行指令

```bash
go test ./internal/service -race -v -run '^TestConcurrentCheckInCreatesSingleAttendance$' -count=20
```

## 错误信息

含缺陷版本中，目标测试会报告实际出勤记录数多于一条，例如：

## 错误堆栈

```text
--- FAIL: TestConcurrentCheckInCreatesSingleAttendance
    attendance_contract_test.go:108: expected one attendance record after concurrent check-in, got 2
FAIL
FAIL	course-attendance/internal/service
```

## 根因

internal/handler/attendance_handler.go 的签到入口、internal/service/attendance_service.go 的重复检查与写入、internal/repository/attendance_repository.go 的身份查询，以及 internal/models/attendance.go 的唯一性定义共同决定一次签到。当前唯一性把签到方式纳入身份键，扫码和补签可以分别成为合法记录；同时服务层采用先查询再写入的分离流程，两个并发请求可能在对方写入前都通过检查。数据库没有为同一课程和学生建立不依赖签到方式的原子唯一边界，统计层又按记录计数，所以重复记录会继续变成重复统计。

生产符号：签到入口、AttendanceService 的重复检查与写入、AttendanceRepository 的查询和 Attendance 模型唯一性共同决定一次签到。具体涉及 AttendanceRepository 的 Create，以及上述链路中的身份查询和唯一性定义。

可靠修复必须让同一课程和学生的记录身份与签到方式解耦，并由数据库唯一约束配合原子写入裁定并发结果；服务层还需要把重复写入转换为稳定的业务结果，不能只在前端去重或只在统计阶段消除重复。
