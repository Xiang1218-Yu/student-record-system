# Bug 复现说明

## Bug 是什么

预置学生账号认领时，外部提交的高权限角色被写入持久化用户并进入后续令牌，学生账号因此获得管理员身份并能访问管理员页面。

## 如何触发

在含 Bug 的环境中执行定向测试。测试通过账号认领服务建立预置学生身份，提交高权限角色，再检查认领后的持久化角色。

## 运行指令

```bash
go test -v ./internal/service -run '^TestClaimedStudentCannotEscalateRole$' -count=1
```

## 错误堆栈

以下错误信息来自上述运行指令在含 Bug 环境中的原始失败输出。

```text
=== RUN   TestClaimedStudentCannotEscalateRole
    claim_contract_test.go:43: claim changed student role to "admin"
--- FAIL: TestClaimedStudentCannotEscalateRole (0.06s)
FAIL
FAIL	course-attendance/internal/service	0.725s
```

## 根因

internal/handler/auth_handler.go 的注册和认领输入、internal/service/auth_service.go 的账号认领编排、internal/models/user.go 的角色持久化，以及 internal/middleware/auth.go 的令牌角色授权共同构成角色信任链。账号认领流程直接信任了外部传入的角色值，没有在认领边界将预置学生身份锁定为学生；该值随后被保存到用户并由令牌授权链读取，形成从注册输入到管理员权限的跨层越权。

生产符号：注册输入解析、AuthService 的 claimAccount、User 模型角色字段和 middleware 的令牌角色判断共同决定角色是否能越过预置账号认领边界。
