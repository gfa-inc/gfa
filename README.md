<div align="center">

# 🚀 GFA

**G**o **F**ramework for **A**pplications

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Gin Framework](https://img.shields.io/badge/Gin-1.11-00ADD8?style=flat)](https://github.com/gin-gonic/gin)

一个开箱即用的 Go 企业级 Web 应用框架，集成丰富的中间件和云原生组件

[English](README_EN.md) | 简体中文

</div>

---

## ✨ 特性

### 🎯 核心功能

- **🏗️ 模块化架构** - 清晰的分层设计，易于扩展和维护
- **⚡ 高性能** - 基于 Gin 框架，轻量级且高效
- **🔧 开箱即用** - 预配置常用中间件和组件
- **🎨 统一响应** - 标准化 API 响应格式，支持链路追踪
- **🛡️ 多层安全** - JWT、API Key、Session 多重认证方式
- **📊 完善日志** - 基于 Zap 的高性能日志系统，支持上下文追踪
- **🔄 优雅关闭** - 支持信号处理和异步任务等待

### 🧩 集成组件

| 组件 | 描述 | 文档 |
|------|------|------|
| 🗄️ **MySQL** | GORM ORM，支持多数据源 | [→](common/db) |
| 💾 **Redis** | 缓存和会话存储，支持泛型 | [→](common/cache) |
| 🔍 **Elasticsearch** | 搜索和分析引擎 (v8) | [→](common/nsdb) |
| 📨 **Kafka** | 消息队列，支持 SASL 认证 | [→](common/mq) |
| ☁️ **AWS S3** | 对象存储服务 | [→](common/aws) |
| 📧 **SMTP** | 邮件发送服务 | [→](common/messenger) |
| 🔐 **Casbin** | RBAC/ABAC 权限管理 | [→](common/casbinx) |
| ⏰ **Cron** | 定时任务调度 | [→](common/sched) |
| 📝 **Swagger** | API 文档自动生成 | [→](common/swag) |
| ✅ **Validator** | 数据验证，支持多语言 | [→](common/validatorx) |

---

## 📦 快速开始

### 安装

```bash
go get github.com/gfa-inc/gfa@latest
```

### 最小示例

```go
package main

import (
    "github.com/gfa-inc/gfa"
    "github.com/gfa-inc/gfa/core"
    "github.com/gin-gonic/gin"
)

type HelloController struct{}

func (h *HelloController) Setup(r *gin.RouterGroup) {
    r.GET("/hello", func(c *gin.Context) {
        core.OK(c, gin.H{"message": "Hello, GFA!"})
    })
}

func main() {
    gfa.Default()                      // 初始化框架
    gfa.AddController(&HelloController{}) // 注册控制器
    gfa.Run()                          // 启动服务
}
```

### 运行项目

```bash
# 1. 创建配置文件 application.yml
# 2. 运行应用
go run main.go

# 3. 访问 API
curl http://127.0.0.1:8888/api/v1/hello
```

---

## 🎨 架构设计

### 目录结构

```
gfa/
├── 📂 core/              # 核心接口和类型定义
│   ├── controller.go     # Controller 接口
│   ├── response.go       # 统一响应格式
│   └── error.go          # 错误类型定义
│
├── 📂 common/            # 通用组件库
│   ├── config/           # 配置管理 (YAML/ENV)
│   ├── logger/           # 日志系统 (Zap)
│   ├── cache/            # 缓存管理 (Redis)
│   ├── db/               # 数据库管理 (MySQL)
│   ├── mq/               # 消息队列 (Kafka)
│   ├── nsdb/             # NoSQL (Elasticsearch)
│   ├── aws/              # AWS 服务 (S3)
│   ├── messenger/        # 消息服务 (邮件)
│   ├── casbinx/          # 权限管理
│   ├── validatorx/       # 数据验证
│   ├── sched/            # 任务调度
│   └── swag/             # API 文档
│
├── 📂 middlewares/       # HTTP 中间件
│   ├── onerror.go        # 错误处理
│   ├── requestid/        # 请求追踪 ID
│   ├── accesslog/        # 访问日志
│   ├── security/         # 安全认证 (JWT/API Key)
│   └── session/          # Session 管理
│
├── 📂 utils/             # 工具函数
│   ├── httpmethod/       # HTTP 方法常量
│   ├── router/           # 路由匹配
│   ├── syncx/            # 并发同步
│   └── ...
│
└── 📂 resources/         # 静态资源
    └── casbin/           # 权限模型文件
```

### 中间件链

```
Request → Recovery → OnError → RequestID → AccessLog
       → Session → Security → Custom Middlewares → Handler
```

---

## 🔧 配置说明

### application.yml 示例

```yaml
server:
  addr: "127.0.0.1:8888"         # 服务地址
  base_path: "/api/v1"           # 基础路径

logger:
  level: debug                    # 日志级别
  ctx_key_mapping:                # 上下文字段映射
    "clientIP": "clientIP"

mysql:
  default:
    dns: "user:pass@tcp(127.0.0.1:3306)/db?charset=utf8mb4&parseTime=True&loc=Local"
    default: true
    level: "debug"

redis:
  default:
    addrs:
      - "127.0.0.1:6379"
    password: "***"
    default: true

elastic:
  default:
    addrs:
      - "http://127.0.0.1:9200"
    default: true

kafka:
  default:
    brokers:
      - "127.0.0.1:9092"
    topic: "gfa"
    default: true

security:
  jwt:
    private_key: "your-secret-key"
  api_key:
    enable: true
    lookup: "header: X-API-KEY, query: token"

session:
  private_key: "session-secret"
  max_age: 86400
  redis:
    addrs:
      - "127.0.0.1:6379"
    password: "***"
```

---

## 📖 使用指南

### 1️⃣ 定义 Controller

```go
type UserController struct{}

func (uc *UserController) Setup(r *gin.RouterGroup) {
    r.GET("/users", uc.List)
    r.POST("/users", uc.Create)
    r.GET("/users/:id", uc.Get)
}

func (uc *UserController) List(c *gin.Context) {
    users := []User{ /* ... */ }
    core.OK(c, core.Paginated(users, 100))
}

func (uc *UserController) Create(c *gin.Context) {
    var user User
    if err := c.ShouldBindJSON(&user); err != nil {
        core.Fail(c, "PARAM_ERROR", err.Error())
        return
    }
    // 业务逻辑...
    core.OK(c, user)
}
```

### 2️⃣ 路由分组

```go
// 按模块分组
gfa.AddGroupControllers("/api/users",
    &UserController{},
    &UserProfileController{},
)

gfa.AddGroupControllers("/api/products",
    &ProductController{},
)
```

### 3️⃣ 自定义中间件

```go
func CustomMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 前置处理
        logger.Info("Before request")

        c.Next()

        // 后置处理
        logger.Info("After request")
    }
}

// 注册中间件
gfa.WithMiddleware(CustomMiddleware())
```

### 4️⃣ 使用数据库

```go
import "github.com/gfa-inc/gfa/common/db"

func main() {
    gfa.Default()

    // 获取默认数据库实例
    database := db.MustDefault()

    // GORM 操作
    var users []User
    database.Find(&users)
}
```

### 5️⃣ 使用缓存

```go
import "github.com/gfa-inc/gfa/common/cache"

func GetUser(id string) (*User, error) {
    rdb := cache.MustDefault()

    // 泛型支持
    user, err := cache.RGet[User](rdb, "user:"+id)
    if err == nil {
        return user, nil
    }

    // 缓存未命中，从数据库加载
    user = loadUserFromDB(id)
    cache.RSet(rdb, "user:"+id, user, time.Hour)

    return user, nil
}
```

### 6️⃣ 安全认证

```go
import "github.com/gfa-inc/gfa/middlewares/security"

func init() {
    // 配置公开路由（无需认证）
    security.PermitRoute("/api/login", httpmethod.MethodPost)
    security.PermitRoute("/api/register", httpmethod.MethodPost)
    security.PermitRoute("/api/health", httpmethod.MethodGet)
}

// JWT Token 生成
func Login(c *gin.Context) {
    token, _ := security.GenerateToken(userID, claims)
    core.OK(c, gin.H{"token": token})
}
```

### 7️⃣ 异步任务

```go
func main() {
    gfa.Default()
    gfa.AddController(&Controller{})

    // 启动异步任务
    gfa.Async(func() {
        // 后台任务逻辑
        processData()
    })

    // 支持取消的异步任务
    gfa.AsyncWithCancel(func(ctx context.Context) {
        for {
            select {
            case <-ctx.Done():
                return
            default:
                // 处理逻辑
            }
        }
    })

    gfa.Run() // 会等待所有异步任务完成
}
```

### 8️⃣ 错误处理

```go
import "github.com/gfa-inc/gfa/core"

func Handler(c *gin.Context) {
    // 参数错误
    if invalidParam {
        panic(core.ParamErr("invalid parameter"))
    }

    // 业务错误
    if businessError {
        panic(core.BizErr("business logic failed"))
    }

    // 未认证
    if !authenticated {
        panic(core.UnauthorizedErr("please login first"))
    }

    // 授权失败
    if !authorized {
        panic(core.AuthErr("permission denied"))
    }
}
```

---

## 🔌 高级功能

### 自定义初始化

```go
gfa.WithSetup(func() {
    // 在组件初始化之前执行
    logger.Info("Pre-setup initialization")
})

gfa.WithPostSetup(func() {
    // 在组件初始化之后执行
    logger.Info("Post-setup initialization")
})

gfa.Default()
gfa.Run()
```

### 配置管理

```go
import "github.com/gfa-inc/gfa/common/config"

// 自定义配置路径
gfa.WithCfgOption(
    config.WithPath("./configs"),
    config.WithPath("/etc/myapp"),
)

// 读取配置
serverAddr := config.GetString("server.addr")
debug := config.GetBool("debug")
maxConn := config.GetInt("database.max_connections")
```

### 日志管理

```go
import "github.com/gfa-inc/gfa/common/logger"

// 上下文日志
logger.Info("User logged in",
    logger.Field("userID", userID),
    logger.Field("ip", clientIP),
)

// 错误日志
logger.Error(err,
    logger.Field("action", "create_user"),
)

// 上下文感知日志
logger.InfoContext(ctx, "Processing request")
```

### Swagger 文档

```go
// 在 main.go 中添加注释

// @title GFA API
// @version 1.0
// @description 企业级 Go Web 应用框架
// @host 127.0.0.1:8888
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-KEY

func main() {
    gfa.Default()
    gfa.Run()
}

// 访问文档: http://127.0.0.1:8888/api/v1/swagger/index.html
```

---

## 🧪 测试

```go
func TestController(t *testing.T) {
    // 初始化测试环境
    gfa.Test(t)

    // 注册测试用的日志核心
    logger.RegisterCore("test", yourTestCore)

    // 测试逻辑
    assert.NotNil(t, config.GetString("server.addr"))
}
```

---

## 🌍 生态系统

### 依赖项

| 库 | 版本 | 用途 |
|----|------|------|
| [Gin](https://github.com/gin-gonic/gin) | 1.11.0 | Web 框架 |
| [GORM](https://gorm.io) | 1.30.1 | ORM |
| [Zap](https://github.com/uber-go/zap) | 1.27.0 | 日志库 |
| [Redis](https://github.com/redis/go-redis) | 9.14.0 | Redis 客户端 |
| [Kafka](https://github.com/segmentio/kafka-go) | 0.4.48 | Kafka 客户端 |
| [Elasticsearch](https://github.com/elastic/go-elasticsearch) | 8.19.0 | ES 客户端 |
| [Casbin](https://github.com/casbin/casbin) | 2.118.0 | 权限管理 |
| [AWS SDK](https://github.com/aws/aws-sdk-go-v2) | 1.38.0 | AWS 服务 |
| [Sonic](https://github.com/bytedance/sonic) | 1.14.0 | JSON 序列化 |

---

## 📈 性能优化

- ⚡ 使用 Sonic 进行高性能 JSON 序列化
- 💾 Redis 连接池优化
- 🔄 数据库连接池配置
- 📊 结构化日志，降低 I/O 开销
- 🎯 中间件链优化，减少不必要的处理

---

## 🤝 贡献

欢迎贡献！请遵循以下步骤：

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

---

## 📝 许可证

本项目基于 MIT 许可证开源 - 详见 [LICENSE](LICENSE) 文件

---

## 🔗 相关链接

- 📖 [文档](https://github.com/gfa-inc/gfa/wiki)
- 🐛 [问题反馈](https://github.com/gfa-inc/gfa/issues)
- 💬 [讨论区](https://github.com/gfa-inc/gfa/discussions)

---

<div align="center">

**如果觉得有帮助，请给一个 ⭐ Star 支持一下！**

Made with ❤️ by GFA Team

</div>
