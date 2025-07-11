# CCany 

**中文** | [English](README-en.md)

一个用Go语言重写的Claude Code代理服务器，提供完整的前后端界面，支持将Claude API请求转换为OpenAI API调用。

![CCany](demo.png)

## 特性

- **完整的Claude API兼容性**: 支持完整的 `/v1/messages` 端点
- **多提供商支持**: OpenAI、Azure OpenAI、本地模型（Ollama）和任何OpenAI兼容的API
- **智能模型映射**: 通过环境变量配置大小模型
- **函数调用**: 完整的工具使用支持和转换
- **流式响应**: 实时SSE流式支持
- **图像支持**: Base64编码图像输入
- **Web管理界面**: 苹果设计风格的管理面板
- **数据库支持**: 使用ent和SQLite3进行数据持久化
- **用户管理**: 完整的用户认证和授权系统
- **请求日志**: 详细的API请求日志记录和分析
- **系统监控**: 实时系统性能监控和健康检查
- **缓存优化**: 智能缓存系统提升性能
- **错误处理**: 全面的错误处理和日志记录

## 快速开始

### 1. 安装依赖

```bash
# 确保已安装Go 1.21+
go version

# 下载依赖
go mod tidy
```

### 2. 可选环境变量配置

```bash
# 复制配置文件（可选）
cp .env.example .env

# 编辑 .env 文件设置数据存储目录和安全密钥
# 所有API和服务配置都通过Web界面管理
```

### 3. 启动服务器

```bash
# 直接运行
go run cmd/server/main.go

# 或者构建后运行
go build -o ccany cmd/server/main.go
./ccany
```

### 4. 初始化设置

```bash
# 启动服务器后，首次访问需要进行初始化设置
# 访问 http://localhost:8082/setup 创建管理员账户并配置API密钥
# 或者使用部署脚本进行自动化部署：
chmod +x scripts/deploy.sh
./scripts/deploy.sh start
```

### 5. 配置API密钥

```bash
# 登录Web界面后，在管理面板中配置：
# - OpenAI API密钥和基础URL
# - Claude API密钥和基础URL
# - 模型配置
# - 性能参数
```

### 6. 使用Claude Code

```bash
ANTHROPIC_BASE_URL=http://localhost:8082 ANTHROPIC_AUTH_TOKEN="some-api-key" claude
```

### 7. 访问Web界面

打开浏览器访问 `http://localhost:8082` 查看管理面板。

## 配置

### 环境变量（可选）

**系统配置:**

- `CLAUDE_PROXY_DATA_PATH` - 数据存储目录 (默认: `./data`)
- `CLAUDE_PROXY_MASTER_KEY` - 主密钥用于加密敏感配置 (生产环境建议设置)
- `JWT_SECRET` - JWT密钥用于用户认证 (生产环境建议设置)

### 后台配置（通过Web界面管理）

**API配置:**

- OpenAI API密钥和基础URL
- Claude API密钥和基础URL
- Azure API版本（可选）

**模型配置:**

- 大模型 (默认: `gpt-4o`)
- 小模型 (默认: `gpt-4o-mini`)

**服务器设置:**

- 服务器主机 (默认: `0.0.0.0`)
- 服务器端口 (默认: `8082`)
- 日志级别 (默认: `info`)

**性能配置:**

- 最大Token限制 (默认: `4096`)
- 最小Token限制 (默认: `100`)
- 请求超时秒数 (默认: `90`)
- 最大重试次数 (默认: `2`)
- 温度参数 (默认: `0.7`)
- 流式响应 (默认: `true`)

> 💡 **注意**: 所有API和服务配置现在都通过Web管理界面进行配置，不再需要环境变量。首次运行时访问 `/setup` 页面进行初始化设置。

### 模型映射

代理将Claude模型请求映射到你配置的模型:

| Claude请求                     | 映射到        | 环境变量       |
| ------------------------------ | ------------- | -------------- |
| 包含"haiku"的模型              | `SMALL_MODEL` | 默认: `gpt-4o-mini` |
| 包含"sonnet"或"opus"的模型     | `BIG_MODEL`   | 默认: `gpt-4o` |

### 提供商配置示例

#### OpenAI
通过Web界面配置：
- OpenAI API密钥：`sk-your-openai-key`
- OpenAI基础URL：`https://api.openai.com/v1`
- 大模型：`gpt-4o`
- 小模型：`gpt-4o-mini`

#### Azure OpenAI
通过Web界面配置：
- OpenAI API密钥：`your-azure-key`
- OpenAI基础URL：`https://your-resource.openai.azure.com/openai/deployments/your-deployment`
- Azure API版本：`2024-02-01`
- 大模型：`gpt-4`
- 小模型：`gpt-35-turbo`

#### 本地模型 (Ollama)
通过Web界面配置：
- OpenAI API密钥：`dummy-key`（必需但可以是虚拟的）
- OpenAI基础URL：`http://localhost:11434/v1`
- 大模型：`llama3.1:70b`
- 小模型：`llama3.1:8b`

#### Claude官方API
通过Web界面配置：
- Claude API密钥：`sk-ant-your-claude-key`
- Claude基础URL：`https://api.anthropic.com`

## Web管理界面

访问 `http://localhost:8082` 可以使用Web管理界面，包含以下功能:

- **仪表板**: 查看服务状态和统计信息
- **请求日志**: 查看所有API请求的详细记录和分析
- **系统监控**: 实时系统性能监控和健康检查
- **用户管理**: 用户账户管理和权限控制
- **配置管理**: 查看和修改系统配置参数
- **API测试**: 测试连接和发送测试消息

界面采用苹果设计风格，支持响应式布局和磨砂玻璃效果。完整的认证系统确保管理界面的安全性。

## 项目结构

```
ccany/
├── cmd/
│   └── server/
│       └── main.go              # 主程序入口
├── internal/
│   ├── app/                     # 应用配置管理
│   ├── auth/                    # 认证服务
│   ├── cache/                   # 缓存服务
│   ├── claudecode/              # Claude Code兼容性服务
│   ├── client/                  # OpenAI客户端
│   ├── config/                  # 配置管理
│   ├── converter/               # 请求/响应转换器
│   ├── crypto/                  # 加密服务
│   ├── database/                # 数据库管理
│   ├── handlers/                # HTTP处理器
│   ├── logging/                 # 请求日志记录
│   ├── middleware/              # 中间件
│   ├── models/                  # 数据模型
│   └── monitoring/              # 系统监控
├── ent/
│   ├── schema/                  # 数据库模式定义
│   └── ...                     # 生成的ORM代码
├── tests/
│   └── basic_test.go           # 基础测试文件
├── scripts/
│   └── deploy.sh                # 部署脚本
├── web/
│   ├── index.html              # 主页面
│   ├── setup.html              # 设置页面
│   └── static/
│       ├── css/                # 样式文件
│       └── js/                 # JavaScript文件
├── .env.example                # 配置模板
├── go.mod                      # Go模块文件
├── Makefile                    # 构建脚本
└── README.md                   # 本文件
```

## API端点

### Claude兼容端点

- `POST /v1/messages` - 创建消息
- `POST /v1/messages/count_tokens` - 计算Token数量

### 管理端点

- `GET /` - Web管理界面
- `GET /health` - 健康检查
- `GET /ready` - 就绪检查
- `GET /setup` - 初始化设置界面
- `POST /setup` - 提交初始化设置

### 认证端点

- `POST /admin/auth/login` - 管理员登录
- `POST /admin/auth/logout` - 管理员登出
- `GET /admin/auth/profile` - 获取用户信息

### 管理API端点

- `GET /admin/users` - 获取用户列表
- `POST /admin/users` - 创建用户
- `PUT /admin/users/:id` - 更新用户
- `DELETE /admin/users/:id` - 删除用户
- `GET /admin/config` - 获取配置信息
- `PUT /admin/config` - 更新配置
- `GET /admin/request-logs` - 获取请求日志
- `GET /admin/request-logs/stats` - 获取请求统计
- `DELETE /admin/request-logs` - 清理请求日志

### 监控端点

- `GET /admin/monitoring/health` - 系统健康检查
- `GET /admin/monitoring/metrics` - 系统指标
- `GET /admin/monitoring/system` - 系统信息

## 开发

### 构建

```bash
# 构建可执行文件
go build -o ccany cmd/server/main.go

# 交叉编译
GOOS=linux GOARCH=amd64 go build -o ccany-linux cmd/server/main.go
GOOS=windows GOARCH=amd64 go build -o ccany.exe cmd/server/main.go
```

### 测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/converter

# 运行集成测试（需要先启动服务器）
go test -v ./tests/

# 运行基准测试
go test -bench=. ./tests/

# 生成测试覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### 代码格式化

```bash
# 格式化代码
go fmt ./...

# 检查代码
go vet ./...
```

## 性能

- **并发处理**: 使用Goroutines实现高并发
- **连接池**: 高效的HTTP连接管理
- **流式支持**: 实时响应流
- **智能缓存**: 多策略缓存系统（LRU、LFU、TTL）
- **请求日志**: 异步日志记录不影响性能
- **系统监控**: 低开销的实时性能监控
- **数据库优化**: 连接池和查询优化
- **内存管理**: 优雅的内存使用和垃圾回收
- **可配置超时**: 可配置的超时和重试
- **智能错误处理**: 详细的日志记录

### 与Claude Code集成

此代理设计为与Claude Code CLI无缝协作。**增强版本包含完整的Claude Code兼容性支持**:

```bash
# 使用部署脚本启动增强版代理
./scripts/deploy.sh start

# 使用Claude Code与代理
ANTHROPIC_BASE_URL=http://localhost:8082 claude

# 或永久设置
export ANTHROPIC_BASE_URL=http://localhost:8082
claude
```

### 增强版Claude Code特性

- ✅ **完整的SSE事件序列**: 支持 `message_start`, `content_block_start`, `ping`, `content_block_delta`, `content_block_stop`, `message_delta`, `message_stop` 事件
- ✅ **请求取消支持**: 客户端断开检测和优雅的请求取消
- ✅ **Claude配置自动化**: 自动创建 `~/.claude.json` 配置文件
- ✅ **Thinking模式**: 支持 `thinking` 字段和智能模型路由
- ✅ **增强工具调用**: 支持增量JSON解析的工具调用流式传输
- ✅ **缓存Token**: 支持 `cache_read_input_tokens` 使用报告
- ✅ **智能路由**: 基于复杂度和token数量的智能模型选择

### 部署选项

```bash
# 基本部署
./scripts/deploy.sh start

# 包含监控的部署
./scripts/deploy.sh monitoring

# 包含Nginx的部署
./scripts/deploy.sh nginx

# 测试Claude Code兼容性
./scripts/deploy.sh test

# 显示帮助
./scripts/deploy.sh help
```

## Docker部署

### 使用Docker Compose

```bash
# 复制环境配置
cp .env.example .env
# 编辑 .env 文件配置API密钥

# 基本部署
docker-compose up -d

# 包含监控的部署
docker-compose --profile monitoring up -d

# 包含Nginx的部署
docker-compose --profile nginx up -d

# 测试Claude Code兼容性
docker-compose --profile test up --build test-claude-code
```

### 使用部署脚本

```bash
# 自动化部署（推荐）
./scripts/deploy.sh start

# 包含监控栈的部署
./scripts/deploy.sh monitoring

# 检查服务状态
./scripts/deploy.sh status

# 查看日志
./scripts/deploy.sh logs
```

详细的部署指南请参考：[部署文档](docs/DEPLOYMENT_GUIDE.md)

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request！

## 更新日志

### v1.3.0 (增强版 - Claude Code兼容性)
- ✅ 完整的Claude Code兼容性支持
- ✅ 增强的SSE事件序列 (message_start, content_block_start, ping, content_block_delta, content_block_stop, message_delta, message_stop)
- ✅ 请求取消和客户端断开检测
- ✅ Claude配置自动初始化 (~/.claude.json)
- ✅ Thinking模式支持和智能模型路由
- ✅ 增强的工具调用流式传输
- ✅ 缓存Token使用报告
- ✅ Docker和Docker Compose增强配置
- ✅ GitHub Actions CI/CD流水线
- ✅ 完整的部署脚本和监控支持
- ✅ 增强的Nginx配置和性能优化

### v1.2.0
- 完整的后台管理系统
- 用户认证和授权
- 请求日志记录和分析
- 系统监控和健康检查
- 智能缓存系统
- 完整的测试套件
- 增强的文档和部署指南

### v1.1.0
- 数据库集成和ORM支持
- 配置管理系统
- 安全的配置存储
- 数据库迁移支持

### v1.0.0
- 初始Go版本发布
- 完整的Claude API兼容性
- Web管理界面
- 数据库支持
- 苹果设计风格UI
