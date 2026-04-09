# CSU-Star Backend

中南大学课程资源共享与评价平台后端服务。

## 项目简介

CSU-Star 是一个面向高校师生的课程资源分享与评价的一站式综合平台，提供课程资料上传下载、课程与教师评价、社交互动等功能。

## 技术栈

- **语言**: Go 1.26
- **框架**: Gin Web Framework
- **数据库**: PostgreSQL + GORM
- **缓存**: Redis
- **文件存储**: 腾讯云 COS
- **邮件服务**: 腾讯云 SES
- **认证**: JWT + OAuth2

## 项目结构

```
csu-star-backend/
├── internal/
│   ├── handler/     # HTTP 处理器
│   ├── middleware/  # 中间件 (JWT认证、角色权限、CORS等)
│   ├── model/       # 数据模型
│   ├── req/         # 请求结构体
│   ├── resp/        # 响应结构体
│   ├── repo/        # 数据访问层
│   └── service/     # 业务逻辑层
├── pkg/utils/       # 工具函数
├── router/          # 路由配置
├── database/        # 数据库初始化脚本
├── data/            # 初始数据 (课程、教师、部门)
├── logger/          # 日志配置
└── config.yaml      # 配置文件
```

## 主要功能模块

### 认证模块 (/auth)
- 邮箱注册与登录
- 验证码发送与验证
- 密码找回
- OAuth 登录 (QQ、GitHub、Google)
- Token 刷新与登出

### 院系模块 (/dept)
- 院系列表查询

### 课程模块 (/course)
- 课程信息管理
- 课程评价与评分

### 教师模块 (/teacher)
- 教师信息管理
- 教师评价与排行榜

### 资源模块 (/resource)
- 课程资料上传下载 (PPT、PDF、笔记、试卷、实验等)
- 资源审核流程
- 下载积分系统

### 社交模块 (/social)
- 点赞与收藏
- 关注功能
- 举报管理

### 评论模块 (/comment)
- 课程/教师评论
- 评论回复

### 排行榜模块 (/ranking)
- 教师排行榜
- 课程排行榜

### 通知模块 (/notify)
- 系统通知
- 未读消息计数

### 管理后台 (/admin)
- 用户管理
- 内容审核
- 数据统计

## API 列表

| 模块 | 路径 | 说明 |
|------|------|------|
| Auth | /auth/email/register | 邮箱注册 |
| Auth | /auth/email/login | 邮箱登录 |
| Auth | /auth/email/captcha | 发送验证码 |
| Auth | /auth/oauth/login | OAuth登录 |
| Course | /courses | 课程列表 |
| Course | /courses/:id/evaluations | 课程评价 |
| Teacher | /teachers | 教师列表 |
| Teacher | /teachers/:id/evaluations | 教师评价 |
| Resource | /resources | 资源列表 |
| Resource | /resources/upload | 上传资源 |
| Ranking | /ranking/teachers | 教师排行榜 |
| Ranking | /ranking/courses | 课程排行榜 |

## 配置说明

复制配置文件并修改相应配置：

```bash
cp config-secret.yaml config.yaml
```

主要配置项：

```yaml
server:
  port: 8080

database:
  dsn: "host=localhost user=postgres dbname=csu_star port=5432 sslmode=disable"

redis:
  addr: "localhost:6379"

jwt:
  secret: "your_jwt_secret"
  access_expiration: 3600      # 1小时
  refresh_expiration: 604800   # 7天

tencent:
  secret_id: "your_secret_id"
  secret_key: "your_secret_key"
  ses:
    from_email_addr: "your_email"
  cos:
    bucket: "your_bucket"
    region: "your_region"

oauth:
  qq:
    app_id: "your_qq_app_id"
    app_key: "your_qq_app_key"
  wechat:
    app_id: "your_wechat_app_id"
    app_secret: "your_wechat_secret"
  github:
    client_id: "your_github_client_id"
    client_secret: "your_github_secret"
  google:
    client_id: "your_google_client_id"
    client_secret: "your_google_secret"
```

## 环境要求

- Go 1.25+
- PostgreSQL 14+
- Redis 6+
- 腾讯云账号 (COS + SES)

## 运行

```bash
# 安装依赖
go mod tidy

# 初始化数据库
bash database/init.sh

# 运行服务
go run cmd/main.go
```

## Redis 缓存策略

项目使用 Redis 进行多类数据缓存：

- **会话管理**: Access Token 黑名单、Refresh Token、用户会话
- **计数器**: 下载次数、积分统计
- **排行榜**: 教师/课程排行榜缓存
- **限流控制**: API 请求限流
- **验证码**: 邮箱验证码存储
- **通知**: 未读消息计数

## 数据库模型

核心数据模型：

- `Users` - 用户信息
- `Courses` - 课程信息
- `Teachers` - 教师信息
- `Departments` - 院系信息
- `Resources` - 资源文件
- `CourseEvaluations` - 课程评价
- `TeacherEvaluations` - 教师评价
- `Comments` - 评论
- `Likes` / `Favorites` - 互动数据
- `Notifications` - 通知
- `PointsRecords` - 积分记录
- `Invitations` - 邀请记录

## 积分系统

- 上传资源奖励积分
- 邀请用户奖励积分
- 每日签到奖励积分
- 下载资源消耗积分
