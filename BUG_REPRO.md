# 缺陷复现说明

## Bug 是什么

学生批量导入遇到中途持久化失败时，接口返回失败但已写入的账号仍保留，重试会受到残留数据影响。

## 如何触发

提交包含两名学生的导入请求，并使第二条账号写入返回存储错误。

## 根因

导入流程按单条账号执行写入却没有统一事务边界。后续写入失败时，先前账号已经提交，错误虽被处理器返回，却没有将已提交的数据回滚。

## 运行指令

```text
go test ./internal/handler -v -run '^TestStudentImportRollsBackAndPropagatesStorageFailure$' -count=1
```

## 错误信息

持久化失败后数据库仍保留一个已导入账号。

## 错误堆栈

```text
student_import_contract_test.go:71: partial student import remained after storage failure: 1
--- FAIL: TestStudentImportRollsBackAndPropagatesStorageFailure
```
