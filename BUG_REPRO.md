# Bug 复现说明

## Bug 是什么

学生批量导入收到取消信号后，已经创建的账户没有随批量操作一起回滚，导致客户端看到取消结果但数据库留下部分账户；再次提交同一批数据时会遇到已存在的邮箱。

## 如何触发

在含 Bug 的环境中执行定向测试。测试通过导入入口、批量导入服务和用户写入链路触发请求取消，并检查取消后是否仍有账户残留。

## 运行指令

```bash
go test -v ./internal/handler -run '^TestCanceledImportDoesNotLeavePartialAccounts$' -count=1
```

## 错误堆栈

以下错误信息来自上述运行指令在含 Bug 环境中的原始失败输出。

```text
=== RUN   TestCanceledImportDoesNotLeavePartialAccounts
    import_contract_test.go:72: canceled import left 1 accounts; batch must be atomic
--- FAIL: TestCanceledImportDoesNotLeavePartialAccounts (0.06s)
FAIL
FAIL	course-attendance/internal/handler	0.719s
```

## 根因

internal/handler/enrollment_handler.go 的导入入口、internal/service/enrollment_service.go 的批量编排、internal/repository/user_repository.go 的账户写入，以及 internal/models/user.go 的写入钩子共同决定取消后的提交语义。请求上下文的取消信号没有贯穿并约束整个批量写入边界，逐条创建可以在取消发生后继续完成，服务也没有把已完成的账户写入纳入同一原子提交或回滚语义，最终留下部分账户并污染重试状态。

生产符号：导入入口、EnrollmentService 的批量循环、UserRepository 的账户创建和 User 模型写入钩子共同决定取消是否能阻止后续写入以及是否清理已完成写入。
