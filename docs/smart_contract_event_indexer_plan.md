# Smart Contract Event Indexer - 详细项目计划文档

基于需求文档和 **Mono-repo + Microservices** 架构的完整实施计划。

---

## 📋 项目架构概览

### Microservices 拆分策略

```
mono-repo/
├── services/
│   ├── indexer-service/      # 区块链事件索引服务
│   ├── api-gateway/          # GraphQL/REST API网关
│   ├── query-service/        # 查询优化和缓存服务
│   └── admin-service/        # 管理后台服务
├── shared/
│   ├── proto/                # gRPC协议定义
│   ├── models/               # 共享数据模型
│   ├── utils/                # 通用工具库
│   └── config/               # 共享配置
├── infrastructure/
│   ├── docker/               # Docker配置
│   ├── k8s/                  # Kubernetes manifests
│   └── terraform/            # 基础设施即代码
└── docs/                     # 项目文档
```

### 服务职责划分

| 服务 | 职责 | 技术栈 | 端口 |
|------|------|--------|------|
| **indexer-service** | 监听区块链、解析事件、写入数据库 | Go + go-ethereum | 8080 |
| **api-gateway** | 对外API接口、认证、限流 | Go + gqlgen + Gin | 8000 |
| **query-service** | 查询优化、聚合计算、缓存 | Go + Redis | 8081 |
| **admin-service** | 管理配置、监控面板、健康检查 | Go + React (前端) | 8082 |

---

## 🔄 Git 工作流和提交策略

### 核心原则

**原子化提交**: 每个子任务完成后立即提交，保持提交历史的清晰和可审查性。

**提交格式**: 遵循 Conventional Commits 规范
```bash
type(scope): description

详细描述:
- 具体变更内容
- 影响范围
- 依赖关系

Resolves: Phase X Task Y - 任务描述
```

### 阶段开发模式

每个阶段包含多个子任务，每个子任务完成后独立提交：

```
Phase 3: API Layer Development
├── Task 1: GraphQL Schema Design
│   ├── feat(graphql): design complete GraphQL schema with custom scalars
│   └── feat(graphql): configure gqlgen code generation
├── Task 2: gRPC Service Definitions  
│   ├── feat(grpc): define QueryService proto interface
│   └── feat(grpc): define AdminService proto interface
├── Task 3: Query Service Implementation
│   ├── feat(query-service): implement gRPC server with interceptors
│   ├── feat(query-service): add Redis caching layer
│   ├── feat(query-service): build SQL query optimizer
│   └── feat(query-service): add Prometheus metrics
└── Task 4: API Gateway Implementation
    ├── feat(api-gateway): implement REST API endpoints
    ├── feat(api-gateway): add middleware for CORS and logging
    └── feat(api-gateway): implement health check endpoints
```

### 分支策略

- **功能分支**: `feature/phase-X-description`
- **修复分支**: `fix/description`
- **文档分支**: `docs/description`

### 提交检查清单

提交前必须检查：
- [ ] 运行测试: `make test`
- [ ] 运行代码检查: `make lint`
- [ ] 检查变更: `git diff --cached`
- [ ] 确认提交信息清晰准确
- [ ] 确认包含必要的测试

详细的工作流指南请参考: [Git Workflow Documentation](../development/GIT_WORKFLOW.md)

---

## 🎯 Phase 1: 项目基础设施搭建 (Week 1, Day 1-3)

### 1.1 Mono-repo 初始化

**任务清单:**
- [ ] 创建 mono-repo 目录结构
- [ ] 设置 Go Workspace (go.work) 管理多模块
- [ ] 配置 Makefile 统一构建命令
- [ ] 设置 `.gitignore` 和 `.editorconfig`
- [ ] 初始化版本管理策略 (语义化版本)

**交付物:**
- 完整的项目骨架
- 可运行的 `make build` 命令

---

### 1.2 共享模块开发

**任务清单:**
- [ ] 定义共享数据模型 (Event, Contract, EventArg)
- [ ] 创建 gRPC proto 文件定义服务间接口
- [ ] 实现通用配置加载器 (支持 ENV/YAML)
- [ ] 实现通用日志库 (结构化日志 + 分级)
- [ ] 实现通用错误处理中间件

**交付物:**
- `shared/models` 包
- `shared/proto` gRPC 定义
- `shared/utils` 工具库

---

### 1.3 Docker 开发环境

**任务清单:**
- [ ] 编写 `docker-compose.yml` (PostgreSQL + Redis + 测试网节点)
- [ ] 配置 PostgreSQL 初始化脚本
- [ ] 配置 Redis 持久化
- [ ] 设置本地 Ganache/Hardhat 测试节点
- [ ] 编写服务健康检查脚本

**Docker Compose 服务清单:**
```yaml
services:
  - postgres:15-alpine
  - redis:7-alpine
  - ganache (或 geth-dev)
  - adminer (数据库管理界面)
```

**交付物:**
- 一键启动的本地开发环境
- 数据库迁移脚本 v1 (初始 schema)

---

## 🔧 Phase 2: Indexer Service 核心开发 (Week 1 Day 4-7, Week 2 Day 1-3)

### 2.1 区块链连接模块

**任务清单:**
- [ ] 实现多 RPC 节点管理器
  - 主节点 + Fallback 列表
  - 智能切换逻辑
  - 健康检查机制
- [ ] 实现 WebSocket 订阅管理
  - 断线重连
  - 心跳检测
- [ ] 实现区块监听器
  - 获取最新区块
  - 区块确认逻辑 (12块)
- [ ] 编写单元测试 (Mock RPC)

**关键配置:**
```yaml
rpc:
  primary: "wss://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
  fallback:
    - "https://rpc.ankr.com/eth"
    - "https://cloudflare-eth.com"
  max_retry: 3
  retry_delay: 5s
```

**交付物:**
- 稳定的区块链连接层
- 覆盖率 >70% 的单元测试

---

### 2.2 事件解析模块

**任务清单:**
- [ ] 实现 ABI 解析器
  - 加载合约 ABI
  - 提取 Event 定义
- [ ] 实现 Event Log 解析器
  - 解析 indexed 参数
  - 解析 non-indexed 参数
  - 类型转换 (BigNumber → String)
- [ ] 实现事件过滤器
  - 按合约地址过滤
  - 按事件名称过滤
- [ ] 处理特殊类型
  - Address checksum 转换
  - Bytes 转 Hex
  - Tuple 类型展开
- [ ] **实现确认块检查逻辑**
  - 读取合约的 confirm_blocks 配置
  - 检查事件是否达到确认要求
  - 支持合约级别的不同策略

**测试用例:**
- ERC20 Transfer Event
- ERC721 Transfer Event
- Uniswap Swap Event (复杂参数)

**交付物:**
- Event Parser 核心库
- 支持常见 ERC 标准事件

---

### 2.3 数据持久化模块

**任务清单:**
- [ ] 实现数据库连接池
- [ ] 实现 Contract CRUD 操作
  - 幂等的 AddContract
  - 更新 current_block
- [ ] 实现 Event 批量插入
  - 使用 COPY 协议优化性能
  - 处理冲突 (ON CONFLICT DO NOTHING)
- [ ] 实现事务管理
  - 批量操作原子性
- [ ] 实现数据库迁移管理 (golang-migrate)

**性能优化点:**
- 批处理大小: 100-500 events/batch
- 连接池: Min 5, Max 20

**交付物:**
- 高性能的数据存储层
- 完整的数据库迁移脚本

---

### 2.4 Chain Reorg 处理

**任务清单:**
- [ ] 实现区块状态缓存 (Redis)
  - 缓存最近 100 个区块 hash (足够检测深度 reorg)
- [ ] 实现 Reorg 检测逻辑
  - 对比链上 block hash 与缓存
  - 识别分叉点
- [ ] 实现数据回滚机制
  - 删除 block_number > fork_point 的事件
  - 更新 indexer_state
- [ ] 实现重新索引逻辑
  - 从分叉点重新拉取事件
- [ ] 添加 Reorg 告警

**回滚策略:**
```sql
-- 回滚到指定区块
DELETE FROM events WHERE block_number > $fork_block;
UPDATE contracts SET current_block = $fork_block;
```

**交付物:**
- 健壮的 Reorg 处理机制
- Reorg 监控指标

---

### 2.5 索引器主流程整合

**任务清单:**
- [ ] 实现主索引循环
  - 监听新区块
  - 批量获取事件
  - 解析 + 存储
  - 更新状态
- [ ] 实现断点续传
  - 从 indexer_state 恢复
  - 处理服务重启
- [ ] 实现优雅停机
  - 处理完当前批次再退出
  - 保存状态
- [ ] 实现并发控制
  - 多合约并发索引
  - Goroutine 池管理

**配置参数:**
```yaml
indexer:
  batch_size: 100
  default_confirm_blocks: 6  # 默认平衡模式（可在合约级别覆盖）
  poll_interval: 6s
  max_concurrent_contracts: 5
  
  # 确认策略预设
  confirmation_presets:
    realtime: 1   # 实时模式: ~12秒延迟
    balanced: 6   # 平衡模式: ~72秒延迟（推荐）
    safe: 12      # 安全模式: ~144秒延迟
```

**交付物:**
- 完整的 Indexer Service
- 可监控的指标输出 (Prometheus)

---

## 🌐 Phase 3: API Gateway 开发 (Week 2 Day 4-7)

### 3.1 GraphQL Schema 定义

**任务清单:**
- [ ] 设计 GraphQL Schema
  - Query 类型定义
  - Mutation 类型定义
  - 自定义标量 (DateTime, BigInt)
- [ ] 使用 gqlgen 生成代码
- [ ] 实现 DataLoader 防止 N+1 查询
- [ ] 配置 GraphQL Playground

**核心 Query:**
```graphql
type Query {
  events(filter: EventFilter, pagination: Pagination): EventConnection
  eventsByTransaction(txHash: String!): [Event!]!
  eventsByAddress(address: String!, pagination: Pagination): EventConnection
  contract(address: String!): Contract
  contractStats(address: String!): ContractStats
}

type Mutation {
  addContract(input: AddContractInput!): AddContractPayload!
  removeContract(address: String!): RemoveContractPayload!
  triggerBackfill(address: String!, fromBlock: Int!, toBlock: Int!): BackfillPayload!
}
```

**交付物:**
- 完整的 GraphQL Schema
- 自动生成的 Resolver 框架

---

### 3.2 gRPC 客户端实现

**任务清单:**
- [ ] 定义服务间 gRPC 接口
- [ ] 实现 API Gateway → Query Service 调用
- [ ] 实现 API Gateway → Admin Service 调用
- [ ] 配置连接池和超时
- [ ] 实现请求重试逻辑

**gRPC 服务定义:**
```protobuf
service QueryService {
  rpc GetEvents(EventQuery) returns (EventResponse);
  rpc GetEventsByAddress(AddressQuery) returns (EventResponse);
  rpc GetContractStats(StatsQuery) returns (StatsResponse);
}
```

**交付物:**
- 完整的 gRPC 客户端封装
- 服务发现机制 (硬编码 → 后续扩展 Consul/etcd)

---

### 3.3 GraphQL Resolver 实现

**任务清单:**
- [ ] 实现 Query Resolvers
  - events: 调用 Query Service
  - eventsByTransaction: 数据库直查
  - eventsByAddress: 调用 Query Service
  - contract: 数据库查询
  - contractStats: 调用 Query Service
- [ ] 实现 Mutation Resolvers
  - addContract: 调用 Admin Service
  - removeContract: 调用 Admin Service
  - triggerBackfill: 调用 Indexer Service
- [ ] 实现 Pagination
  - Cursor-based pagination
  - 计算 totalCount
- [ ] 实现错误处理
  - 统一错误格式
  - gRPC 错误转换

**交付物:**
- 完整功能的 GraphQL API
- Postman/Insomnia 测试集合

---

### 3.4 REST API 实现 (可选)

**任务清单:**
- [ ] 使用 Gin 框架实现 REST 端点
- [ ] 实现以下端点:
  - `GET /api/events`
  - `GET /api/events/:txHash`
  - `GET /api/contracts/:address`
  - `POST /api/contracts`
- [ ] 实现 API 版本控制 (v1)
- [ ] 生成 OpenAPI 文档

**交付物:**
- RESTful API (与 GraphQL 功能对等)
- Swagger UI 文档

---

### 3.5 认证与限流

**任务清单:**
- [ ] 实现 API Key 认证中间件
- [ ] 实现基于 Redis 的限流
  - 按 API Key 限流
  - 按 IP 限流
- [ ] 实现请求日志记录
- [ ] 实现 CORS 配置

**限流策略:**
- 免费: 100 req/min
- Pro: 1000 req/min

**交付物:**
- 安全的 API Gateway
- 限流监控指标

---

## 🚀 Phase 4: Query Service 开发 (Week 3 Day 1-3)

### 4.1 gRPC 服务端实现

**任务清单:**
- [ ] 实现 gRPC Server
- [ ] 实现 QueryService 接口
  - GetEvents
  - GetEventsByAddress
  - GetContractStats
- [ ] 配置服务端拦截器
  - 日志记录
  - Metrics 收集
  - 错误恢复
- [ ] 实现健康检查端点

**交付物:**
- 可独立运行的 Query Service
- gRPC 健康检查

---

### 4.2 查询优化层

**任务清单:**
- [ ] 实现智能查询路由
  - 简单查询 → 直接数据库
  - 复杂查询 → 优化后执行
- [ ] 实现 `eventsByAddress` 优化
  - MVP: GIN 索引查询
  - 评估是否需要 event_addresses 表
- [ ] 实现查询计划分析
  - 记录慢查询
  - 自动建议索引
- [ ] 实现查询超时控制

**查询性能目标:**
- P50 < 50ms
- P95 < 200ms
- P99 < 500ms

**交付物:**
- 高性能查询引擎
- 慢查询告警

---

### 4.3 缓存层实现

**任务清单:**
- [ ] 实现 Redis 缓存策略
  - 热点查询缓存 (TTL 30s)
  - 合约统计缓存 (TTL 5min)
- [ ] 实现缓存 Key 设计
  - 包含查询参数哈希
  - 版本号机制
- [ ] 实现缓存失效策略
  - 新事件写入时主动失效
  - LRU 淘汰
- [ ] 实现缓存穿透保护
  - 布隆过滤器
  - 空结果缓存

**缓存命中率目标:**
- 热点查询: >80%
- 普通查询: >50%

**交付物:**
- 完善的缓存系统
- 缓存监控面板

---

### 4.4 聚合计算模块

**任务清单:**
- [ ] 实现 ContractStats 计算
  - totalEvents
  - latestBlock
  - indexerDelay
  - uniqueAddresses (如果适用)
- [ ] 实现自定义聚合查询
  - 按时间范围统计
  - 按地址统计
  - Top N 查询
- [ ] 实现聚合结果缓存
- [ ] 实现增量计算优化

**交付物:**
- 强大的聚合查询能力
- 聚合查询文档

---

## 🎛️ Phase 5: Admin Service 开发 (Week 3 Day 4-7)

### 5.1 管理 API 实现

**任务清单:**
- [ ] 实现合约管理接口
  - 添加监听合约 (幂等)
  - 删除监听合约
  - 更新合约配置
  - 列出所有合约
- [ ] 实现索引控制接口
  - 触发历史回填
  - 暂停/恢复索引
  - 重置索引状态
- [ ] 实现系统状态接口
  - 获取索引器状态
  - 获取错误日志
  - 获取性能指标
- [ ] 实现管理员认证
  - JWT Token
  - 角色权限控制

**交付物:**
- 完整的管理 API
- Admin API 文档

---

### 5.2 历史数据回填功能

**任务清单:**
- [ ] 实现 Backfill 任务调度
  - 队列管理 (Redis)
  - 任务状态追踪
- [ ] 实现分块回填逻辑
  - 按区块范围分片
  - 并行处理
  - 进度持久化
- [ ] 实现回填限速
  - 避免 RPC 限流
  - 控制数据库写入速度
- [ ] 实现回填监控
  - 进度百分比
  - ETA 计算
  - 错误重试

**Backfill 配置:**
```yaml
backfill:
  chunk_size: 1000  # 每次处理1000个区块
  max_concurrent: 3
  rate_limit: 100   # 100 req/min
```

**交付物:**
- 可靠的历史数据回填系统
- 回填进度监控

---

### 5.3 错误日志与告警

**任务清单:**
- [ ] 实现错误日志收集
  - 索引器错误
  - API 错误
  - 数据库错误
- [ ] 实现错误分类
  - RPC 错误
  - 解析错误
  - 存储错误
- [ ] 实现告警规则
  - 索引延迟 >1min
  - RPC 连接失败
  - Reorg 检测
- [ ] 集成告警通知
  - Slack Webhook
  - Email (可选)

**交付物:**
- 完善的错误追踪系统
- 实时告警机制

---

### 5.4 管理后台 UI (可选)

**任务清单:**
- [ ] 使用 React + TypeScript 开发
- [ ] 实现页面:
  - Dashboard (系统概览)
  - Contracts (合约管理)
  - Events (事件浏览)
  - Logs (错误日志)
- [ ] 实现数据可视化
  - 索引速度图表 (Chart.js)
  - 事件统计图表
- [ ] 实现实时更新 (WebSocket)

**交付物:**
- 可视化管理后台
- 用户操作手册

---

## 📊 Phase 6: 监控与测试 (Week 4 Day 1-4)

### 6.1 Prometheus Metrics

**任务清单:**
- [ ] 为每个服务添加 Metrics 端点
- [ ] 定义关键指标:
  - **Indexer**: 索引延迟、事件处理速率、RPC 调用次数
  - **API Gateway**: 请求总数、响应时间、错误率
  - **Query Service**: 查询延迟、缓存命中率
- [ ] 实现自定义 Metrics
  - 按合约的事件数量
  - 按事件类型的分布
- [ ] 配置 Prometheus 抓取

**交付物:**
- 完整的 Metrics 采集
- Prometheus 配置文件

---

### 6.2 Grafana Dashboard

**任务清单:**
- [ ] 安装 Grafana
- [ ] 创建 Dashboard:
  - **系统概览**: CPU、内存、网络
  - **索引器性能**: 延迟、吞吐量、Reorg 次数
  - **API 性能**: QPS、延迟分布、错误率
  - **数据库性能**: 连接数、查询时间、慢查询
- [ ] 配置告警规则
  - 索引延迟 >1min 告警
  - API 错误率 >1% 告警
  - 数据库连接池耗尽告警
- [ ] 导出 Dashboard JSON

**交付物:**
- 可视化监控面板
- 告警配置

---

### 6.3 单元测试

**任务清单:**
- [ ] 为每个服务编写单元测试
- [ ] 测试覆盖核心逻辑:
  - Event Parser: 测试各种 ABI 类型
  - Reorg Handler: 模拟 Reorg 场景
  - Query Optimizer: 测试查询性能
- [ ] 使用 Mock
  - Mock RPC 客户端
  - Mock 数据库
  - Mock Redis
- [ ] 配置 CI 自动运行测试

**覆盖率目标:**
- 核心业务逻辑: >80%
- API Handlers: >70%
- 总体: >75%

**交付物:**
- 完整的单元测试套件
- 测试覆盖率报告

---

### 6.4 集成测试

**任务清单:**
- [ ] 使用 Testcontainers 启动依赖
  - PostgreSQL
  - Redis
  - Ganache
- [ ] 编写端到端测试:
  - 部署测试合约
  - 触发事件
  - 验证索引结果
  - 查询 API
- [ ] 测试 Reorg 场景
  - 模拟区块链分叉
  - 验证数据回滚
- [ ] 测试服务间通信
  - gRPC 调用
  - 错误处理

**交付物:**
- E2E 测试套件
- 集成测试文档

---

### 6.5 性能测试

**任务清单:**
- [ ] 使用 k6 编写负载测试
- [ ] 测试场景:
  - **API 压测**: 1000 QPS 持续 5 分钟
  - **批量索引**: 10000 events/批次
  - **并发查询**: 100 并发用户
- [ ] 测试数据库性能
  - 百万级事件查询
  - 复杂聚合查询
- [ ] 性能优化
  - 识别瓶颈
  - 调优参数
  - 再次测试

**性能基准:**
- API P95 < 200ms
- 索引延迟 < 5s
- 数据库查询 P99 < 500ms

**交付物:**
- 性能测试报告
- 优化建议文档

---

## 🚢 Phase 7: 部署与文档 (Week 4 Day 5-7, Week 5)

### 7.1 容器化

**任务清单:**
- [ ] 为每个服务编写 Dockerfile
  - 多阶段构建
  - 最小化镜像大小
  - 非 root 用户运行
- [ ] 编写 docker-compose 生产配置
- [ ] 配置健康检查
- [ ] 配置资源限制

**镜像优化目标:**
- 镜像大小 <50MB (Go binary)
- 构建时间 <3min

**交付物:**
- 生产级 Docker 镜像
- Docker Compose 配置

---

### 7.2 Kubernetes 部署

**任务清单:**
- [ ] 编写 K8s Manifests:
  - Deployment (每个服务)
  - Service (内部/外部)
  - ConfigMap (配置)
  - Secret (敏感信息)
  - HPA (自动扩缩容)
- [ ] 配置 Ingress
  - HTTPS
  - 路由规则
- [ ] 配置持久化存储
  - PostgreSQL PVC
  - Redis PVC
- [ ] 配置 Namespace 隔离

**交付物:**
- 完整的 K8s 部署配置
- 部署脚本 (`make deploy`)

---

### 7.3 CI/CD Pipeline

**任务清单:**
- [ ] 配置 GitHub Actions 或 GitLab CI
- [ ] 实现 CI 流程:
  - 代码检查 (golangci-lint)
  - 运行测试
  - 构建 Docker 镜像
  - 推送到 Registry
- [ ] 实现 CD 流程:
  - 自动部署到 Staging
  - 手动审批部署到 Production
- [ ] 配置回滚机制

**Pipeline 阶段:**
```yaml
stages:
  - lint
  - test
  - build
  - deploy-staging
  - deploy-prod
```

**交付物:**
- 自动化 CI/CD 流程
- Pipeline 配置文件

---

### 7.4 基础设施即代码 (可选)

**任务清单:**
- [ ] 使用 Terraform 定义基础设施
  - AWS ECS/EKS 集群
  - RDS PostgreSQL
  - ElastiCache Redis
  - ALB/NLB
  - VPC 网络
- [ ] 编写环境变量配置
  - Dev
  - Staging
  - Production
- [ ] 实现蓝绿部署策略

**交付物:**
- Terraform 配置文件
- 基础设施文档

---

### 7.5 API 文档

**任务清单:**
- [ ] 编写 GraphQL API 文档
  - Query 说明
  - Mutation 说明
  - 参数类型定义
  - 示例查询
- [ ] 配置 GraphQL Playground
  - 公开访问
  - 添加示例查询
- [ ] 编写 REST API 文档 (如果有)
  - Swagger/OpenAPI 规范
  - 示例请求/响应
- [ ] 编写 gRPC 文档
  - Proto 文件说明
  - 服务间调用示例

**交付物:**
- 完整的 API 参考文档
- Postman Collection

---

### 7.6 架构文档

**任务清单:**
- [ ] 编写架构决策记录 (ADR)
  - 为什么选择微服务
  - 为什么选择 GraphQL
  - 数据库索引策略选择
  - 缓存策略选择
- [ ] 绘制架构图
  - 系统架构图
  - 数据流图
  - 部署架构图
- [ ] 编写运维手册
  - 部署步骤
  - 故障排查
  - 监控指标说明
  - 扩容策略

**交付物:**
- 完整的技术文档
- 可视化架构图

---

### 7.7 用户文档

**任务清单:**
- [ ] 编写 README.md
  - 项目简介
  - 快速开始
  - 功能特性
  - 技术栈
- [ ] 编写开发者指南
  - 本地开发环境搭建
  - 如何添加新合约
  - 如何扩展功能
- [ ] 编写 API 使用教程
  - GraphQL 查询示例
  - 认证说明
  - 限流说明
- [ ] 录制 Demo 视频
  - 系统演示 (5-10分钟)
  - 代码 walkthrough

**交付物:**
- 用户友好的文档
- 视频教程

---

## ✅ 关键检查点 (Checkpoints)

### Week 1 结束
**验收标准:**
- [ ] Docker Compose 环境可一键启动
- [ ] Indexer Service 能监听 Ganache 上的测试合约
- [ ] 事件成功写入 PostgreSQL
- [ ] 能通过 SQL 查询到事件数据

**Demo 演示:**
部署一个 ERC20 合约，执行 Transfer，查看数据库中的记录

---

### Week 2 结束
**验收标准:**
- [ ] GraphQL API 可访问
- [ ] 能通过 GraphQL 查询事件
- [ ] Reorg 处理逻辑通过测试
- [ ] 基础监控指标可见

**Demo 演示:**
使用 GraphQL Playground 执行复杂查询，展示 Reorg 处理

---

### Week 3 结束
**验收标准:**
- [ ] 所有微服务正常运行
- [ ] 服务间 gRPC 通信正常
- [ ] 缓存命中率 >50%
- [ ] API P95 < 300ms (可接受)

**Demo 演示:**
展示管理后台，触发历史回填，查看监控面板

---

### Week 4 结束
**验收标准:**
- [ ] 性能测试通过
- [ ] 单元测试覆盖率 >75%
- [ ] 集成测试全部通过
- [ ] Grafana Dashboard 完善

**Demo 演示:**
展示性能测试结果，模拟高负载场景

---

### Week 5 结束 (项目完成)
**验收标准:**
- [ ] 可部署到 K8s 或云平台
- [ ] CI/CD 流程运行正常
- [ ] 文档完善
- [ ] Demo 视频录制完成

**Portfolio 展示:**
准备好向潜在雇主/客户展示项目

---

## 🎯 成功指标 (Success Metrics)

| 指标 | 目标 | 测量方式 |
|------|------|----------|
| **索引延迟** | 平衡模式: ~72秒, 实时模式: ~12秒, 安全模式: ~144秒 | Prometheus metrics |
| **API 响应时间** | P95 <200ms | k6 压测 |
| **缓存命中率** | >70% | Redis metrics |
| **测试覆盖率** | >75% | go test -cover |
| **系统可用性** | >99% | Uptime monitoring |
| **数据准确率** | 99.99% (6块确认), 99.9999% (12块确认) | 与链上数据对比 |

---

## 🚨 风险缓解计划

### 高优先级风险

**1. RPC 节点不稳定**
- **缓解**: 使用 Alchemy/Infura 免费层 + 3个 fallback 节点
- **监控**: RPC 调用失败率 >5% 告警
- **Portfolio预算**: $0/月 (免费层: Alchemy 300M CU/月 + Infura 100K请求/天)
- **生产预算**: $50-100/月 (如需升级)

**2. Chain Reorg 数据丢失**
- **缓解**: 可配置确认策略（默认6块平衡模式，可选1块实时或12块安全）
- **测试**: 在 Ganache 模拟 Reorg 场景，测试所有确认策略
- **监控**: Reorg 检测告警 + 按确认策略分组的延迟监控

**3. 数据库性能瓶颈**
- **缓解**: 批量插入 + 索引优化 + 连接池
- **监控**: 慢查询日志
- **扩展**: 必要时分表/分库

---

## 📚 推荐学习资源

在开发过程中，您可能需要参考:

- **Go + Ethereum**: https://goethereumbook.org/
- **gqlgen 文档**: https://gqlgen.com/
- **PostgreSQL 性能优化**: https://www.postgresql.org/docs/
- **Kubernetes 实战**: https://kubernetes.io/docs/
- **微服务设计模式**: "Building Microservices" by Sam Newman

---

## 🎉 项目完成后的收获

完成这个项目后，您将能够展示:

✅ **系统设计**: 微服务架构、事件驱动设计
✅ **区块链技术**: Web3 开发、智能合约交互
✅ **后端工程**: Go 微服务、gRPC、GraphQL
✅ **数据工程**: 高性能索引、查询优化
✅ **DevOps**: Docker、K8s、CI/CD
✅ **可观测性**: Prometheus、Grafana 监控

这将是一个非常亮眼的 Portfolio 项目！🚀

---

## 📋 快速任务总览

### Week 1: 基础设施 + Indexer 核心
- Day 1-3: Mono-repo 搭建 + 共享模块 + Docker 环境
- Day 4-7: 区块链连接 + 事件解析 + 数据持久化

### Week 2: Indexer 完善 + API Gateway
- Day 1-3: Reorg 处理 + 索引器主流程整合
- Day 4-7: GraphQL Schema + gRPC 客户端 + Resolver 实现

### Week 3: Query & Admin Services
- Day 1-3: Query Service (gRPC服务端 + 查询优化 + 缓存)
- Day 4-7: Admin Service (管理API + 回填 + 错误日志)

### Week 4: 测试与优化
- Day 1-2: Prometheus + Grafana 监控
- Day 3-4: 单元测试 + 集成测试 + 性能测试

### Week 5: 部署与文档
- Day 1-2: Docker + K8s 部署
- Day 3-4: CI/CD Pipeline + 基础设施即代码
- Day 5-7: API文档 + 架构文档 + 用户文档 + Demo视频

---

**预计总工时**: 160-200 小时  
**建议投入**: 每天 6-8 小时，持续 5 周  
**难度评级**: ⭐⭐⭐⭐☆ (中高级)

---

## 🔮 Future Enhancements (Phase 6+)

以下功能不在当前 Portfolio 项目的 MVP 范围内，但架构已预留扩展空间：

### 1. WebSocket 实时订阅
```graphql
type Subscription {
  newEvents(contractAddress: Address): Event!
}
```
**用途**: 实时推送新事件给客户端  
**复杂度**: 中等 (需要 WebSocket 服务器 + 事件广播机制)  
**优先级**: 低 (Portfolio 展示不必要)

### 2. 多链支持
- 支持 Polygon, BSC, Arbitrum 等 EVM 兼容链
- 统一的多链查询接口
- 按链分离的数据存储

### 3. 高级分析功能
- 时间序列聚合 (每小时/每天统计)
- 地址行为分析 (活跃度、交易模式)
- 智能合约交互图谱

### 4. 企业级功能
- 细粒度权限控制 (RBAC)
- API Key 管理面板
- 数据导出 (CSV/JSON)
- Webhook 通知

**决策**: Portfolio 项目专注于核心功能展示，以上功能可在面试中讨论架构扩展性时提及。

---

**文档版本**: v1.1  
**创建日期**: 2025-10-15  
**更新日期**: 2025-10-17