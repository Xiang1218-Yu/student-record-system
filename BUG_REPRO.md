# Bug 复现说明

## Bug 是什么

移除不存在的活动报名关系时，服务把没有更新任何记录的结果当成成功返回，页面因此显示已移除，但数据库状态没有变化；这会让调用方无法区分真实移除和未命中。

## 如何触发

在含 Bug 的环境中执行定向测试。测试覆盖正常移除、历史记录保留和再次报名，再请求一个不存在的活动报名关系，检查未命中是否被报告为失败。

## 运行指令

```bash
go test -v ./internal/handler -run '^TestEnrollmentRemoveReportsMissAndPreservesLifecycle$' -count=1
```

## 错误堆栈

以下错误信息来自上述运行指令在含 Bug 环境中的原始失败输出。

```text
=== RUN   TestEnrollmentRemoveReportsMissAndPreservesLifecycle
    enrollment_remove_contract_test.go:55: removing a missing active enrollment returned success
--- FAIL: TestEnrollmentRemoveReportsMissAndPreservesLifecycle (0.01s)
FAIL
FAIL	course-attendance/internal/handler	0.618s
```

## 修复结果

Claude 修复了报名移除链路对更新结果的判断和错误传播，使未命中活动报名返回失败，同时保留正常移除、历史记录和再次报名的生命周期语义。修复后同一验证命令稳定通过 5/5 轮。

## 根因

internal/models/enrollment.go 的活动状态语义、internal/repository/enrollment_repository.go 的软删除更新、internal/service/enrollment_service.go 的移除编排，以及 internal/handler/enrollment_handler.go 的响应转换共同决定移除结果。原实现没有把数据库更新的实际影响传递到服务层，未命中活动报名时仍沿用成功路径，造成数据库状态与 HTTP 结果不一致；Claude 的修复边界是保留影响行数并在未命中时传播失败，同时不改变历史记录和再次报名的生命周期。

生产符号：Enrollment 模型的 IsActive、仓储的 Deactivate 更新、EnrollmentService 的移除结果判断和 handler 的失败响应。
