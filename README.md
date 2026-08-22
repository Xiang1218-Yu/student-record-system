# 轻量级线上课程表与考勤记录（Course Attendance System）

面向中小型教育培训机构、企业内部培训或社团活动的课程管理与考勤工具。讲师创建课程、学员查看课程表并扫码签到、系统自动统计出勤率。

后端使用 **Go + Gin + GORM + SQLite**，前端使用 **服务端渲染 HTML 模板 + Tailwind CSS（CDN）+ 少量原生 JS**。代码遵循单一职责原则（Single Responsibility Principle）分层组织。

---

## 功能一览

- 用户认证：邮箱+密码注册/登录，JWT（24h），三种角色（管理员/讲师/学员）
- 课程管理：创建、按周批量创建、编辑、删除、列表分页、按日期筛选
- 学员管理：报名/移除、CSV 或 JSON 批量导入
- 签到：每节课生成带 token 的唯一二维码（默认课程开始前 30 分钟至结束后 30 分钟有效），扫码签到，讲师手动补签，防重复签到
- 出勤统计：个人出勤率、迟到率、单课程出勤率与名单、历史记录
- 数据导出：单课程出勤报表导出为 Excel（.xlsx）

默认管理员账号（首次启动自动创建）：`admin@example.com` / `admin123`

---

## 目录结构（按单一职责分层）

```
course-attendance/
├── main.go                      # 入口：组装 config/db/service/handler/router，不含业务逻辑
├── config/                      # 配置加载（Viper + 环境变量）
├── internal/
│   ├── models/                  # GORM 实体，每个文件只管一个聚合的 schema
│   │   ├── user.go  course.go  enrollment.go  attendance.go  qrcode.go
│   ├── database/                # 数据库连接与 AutoMigrate（唯一持有 *gorm.DB）
│   ├── repository/              # 数据访问层，每个聚合一个文件，只写查询
│   │   ├── user_repository.go  course_repository.go  enrollment_repository.go
│   │   ├── attendance_repository.go  qrcode_repository.go
│   ├── service/                 # 业务逻辑层，编排 repository，不碰 HTTP
│   │   ├── auth_service.go  course_service.go  enrollment_service.go
│   │   ├── attendance_service.go  statistics_service.go
│   │   ├── export_service.go  notification_service.go
│   ├── handler/                 # HTTP 层，每个资源一个文件，仅做 HTTP↔service 转换
│   ├── page/                    # HTML 页面控制器与 cookie 认证
│   ├── middleware/             # JWT 鉴权与角色校验中间件
│   ├── router/                  # 路由表：挂载中间件、绑定路径到 handler
│   └── templates/               # 模板加载器
├── pkg/                         # 可复用工具，无业务依赖
│   ├── hashutil/  jwtauth/  qrutil/  httpx/  logger/
├── web/
│   ├── templates/               # HTML 模板（partials + 页面）
│   └── static/                  # 静态资源（css/js）
└── system.md                    # 原始需求文档
```

### 分层职责

| 层 | 职责 | 不允许做的事 |
|----|------|--------------|
| `models` | 定义表结构与 GORM 钩子 | 不写查询、不碰 HTTP |
| `repository` | 单一聚合的数据访问 | 不含业务规则 |
| `service` | 业务规则与事务编排 | 不写 HTTP 响应、不依赖 gin |
| `handler` / `page` | HTTP↔service 转换 | 不含业务逻辑 |
| `pkg` | 通用工具 | 不依赖 internal |

---

## 技术栈

| 层级 | 选型 |
|------|------|
| 后端框架 | Gin |
| ORM | GORM |
| 数据库 | SQLite（可扩展 Postgres） |
| 前端 | 服务端渲染 HTML + Tailwind CSS（CDN） |
| 二维码 | skip2/go-qrcode |
| Excel 导出 | xuri/excelize |
| 认证 | JWT（golang-jwt/jwt/v5） |
| 密码哈希 | bcrypt（golang.org/x/crypto） |
| 日志 | Zap |
| 配置 | Viper |

---

## 快速开始

### 环境要求
- Go 1.21+

### 安装依赖
```bash
go mod download
```

### 配置
复制环境变量模板并设置密钥：
```bash
cp .env.example .env
# 编辑 .env，将 JWT_SECRET 改为随机长字符串
```

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `SERVER_PORT` | 服务端口 | `8080` |
| `DB_TYPE` | 数据库类型 | `sqlite` |
| `DB_DSN` | 数据库连接串 | `./course.db` |
| `JWT_SECRET` | JWT 密钥（**必填**） | — |
| `BASE_URL` | 服务基础 URL（用于二维码跳转地址） | `http://localhost:8080` |

### 构建与运行
```bash
# 构建
go build -o course-attendance ./main.go

# 运行
./course-attendance
```

或直接：
```bash
go run ./main.go
```

启动后访问 <http://localhost:8080>。

---

## API 概览

所有 JSON 接口前缀 `/api/v1`，除登录/注册与公开只读接口外，需携带 `Authorization: Bearer <token>`。

### 认证
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/register` | 注册 |
| POST | `/api/v1/auth/login` | 登录 |
| GET | `/api/v1/auth/me` | 当前用户信息 |

### 课程
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/courses` | 创建课程（admin/teacher） |
| POST | `/api/v1/courses/batch` | 批量创建（按周重复） |
| GET | `/api/v1/courses` | 课程列表（分页、`?date=` 筛选） |
| GET | `/api/v1/courses/:id` | 课程详情 |
| PUT | `/api/v1/courses/:id` | 更新（仅讲师本人/管理员） |
| DELETE | `/api/v1/courses/:id` | 删除（仅讲师本人/管理员） |

### 讲师
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/teachers` | 讲师列表（分页，**仅 admin**） |
| POST | `/api/v1/teachers` | 新增讲师 `{email,password,name,phone}`（仅 admin） |
| GET | `/api/v1/teachers/:id` | 讲师详情（仅 admin） |
| PUT | `/api/v1/teachers/:id` | 更新姓名/邮箱/手机（仅 admin） |
| PATCH | `/api/v1/teachers/:id/password` | 重置密码 `{new_password}`（仅 admin） |
| DELETE | `/api/v1/teachers/:id` | 删除讲师（仅 admin；名下有活跃课程时返回 409） |

> 讲师即 `users` 表中 `role=teacher` 的用户。密码字段永不返回；`role` 不可经接口改写，恒为 teacher。删除采用软删除（保留历史记录），但若该讲师名下仍有 `scheduled`/`ongoing` 状态的课程则拒绝删除（HTTP 409），需先转派或结课。创建课程表单的讲师下拉由服务端直接读取，不依赖上述公开 API。

### 学员
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/courses/:id/enroll` | 报名 `{student_id}` |
| DELETE | `/api/v1/courses/:id/enroll/:studentId` | 移除学员 |
| GET | `/api/v1/courses/:id/students` | 课程学员名单 |
| POST | `/api/v1/students/import` | 批量导入（JSON `{"students":[...]}` 或表单 `csv`） |

### 签到
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/courses/:id/qr` | 获取签到二维码 token |
| GET | `/api/v1/courses/:id/qr/image` | 获取二维码 PNG 图片 |
| POST | `/api/v1/courses/:id/checkin` | 扫码签到 `{token}` |
| POST | `/api/v1/courses/:id/checkin/manual` | 手动补签（admin/teacher）`{student_id}` |
| GET | `/api/v1/courses/:id/attendance` | 课程签到记录 |

### 统计与导出
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/my-attendance` | 我的出勤统计 |
| GET | `/api/v1/my-attendance/history` | 我的签到历史 |
| GET | `/api/v1/statistics/:courseId` | 课程整体出勤统计 |
| GET | `/api/v1/statistics/export/:courseId` | 导出 Excel |
| GET | `/api/v1/attendance/recent?since=YYYY-MM-DD` | 近期签到记录 |

---

## 前端页面

| 页面 | 路径 |
|------|------|
| 登录/注册 | `/login`, `/register` |
| 课程列表（首页） | `/` |
| 课程详情 | `/courses/:id` |
| 创建课程 | `/courses/new` |
| 我的课表 | `/my-schedule` |
| 签到确认 | `/checkin/:courseId?token=...` |
| 出勤统计 | `/attendance` |
| 学员管理 | `/students` |
| 讲师管理 | `/teachers`（仅 admin） |

---

## 签到状态规则

- **正常 on_time**：课程开始时间前或开始后 30 分钟内签到
- **迟到 late**：开始 30 分钟后签到
- **缺勤 absent**：未签到（统计时按"报名人数 − 已签到"计算）

二维码有效期：课程开始前 30 分钟至结束后 30 分钟；过期后签到会返回"二维码已过期"。签到时间由服务器记录，防止客户端篡改。

---

## 安全说明

- JWT 24 小时过期；密钥由 `JWT_SECRET` 注入
- 密码使用 bcrypt 哈希存储
- 二维码携带唯一 token，服务端校验 token 与课程归属
- 讲师只能编辑/删除自己的课程；学员只能查看与自身相关的数据
- 角色校验通过 `middleware.RequireRole` 强制

---

## 可扩展点

- 数据库：在 `internal/database/database.go` 的 `openDriver` 增加分支即可接入 Postgres
- 通知：`notification_service.go` 目前记录结构化日志，替换其方法体即可接入邮件/微信/IM
- 前端：模板与 `web/static` 可平滑替换为 React SPA（保持现有 API 不变）

---

## 开发说明

```bash
# 编译检查
go build ./...

# 静态检查
go vet ./...
```

需求文档见 [`system.md`](./system.md)。
