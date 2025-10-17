# Smart Contract Event Indexer - 架构设计文档

## 文档信息

| 项目 | Smart Contract Event Indexer |
|------|------------------------------|
| 版本 | v1.0 |
| 作者 | [Your Name] |
| 日期 | 2025-10-15 |
| 状态 | Draft / In Review / Approved |

---

## 目录

1. [系统概览](#1-系统概览)
2. [架构设计原则](#2-架构设计原则)
3. [系统架构](#3-系统架构)
4. [微服务设计](#4-微服务设计)
5. [数据架构](#5-数据架构)
6. [API 设计](#6-api-设计)
7. [技术栈选型](#7-技术栈选型)
8. [关键设计决策](#8-关键设计决策)
9. [部署架构](#9-部署架构)
10. [安全设计](#10-安全设计)
11. [性能优化](#11-性能优化)
12. [可观测性](#12-可观测性)
13. [容错与高可用](#13-容错与高可用)
14. [扩展性设计](#14-扩展性设计)

---

## 1. 系统概览

### 1.1 系统目标

构建一个高性能、可扩展的区块链事件索引系统，实时监听智能合约事件，提供快速查询服务。

**核心价值主张:**
- 🚀 **性能**: 索引延迟 <5秒，API 响应 P95 <200ms
- 🔒 **可靠**: 99.9% 可用性，自动处理链重组
- 📊 **灵活**: GraphQL 支持复杂查询，支持多种聚合分析
- 🔧 **可维护**: 微服务架构，独立部署和扩展

### 1.2 用户场景

```
┌─────────────────┐
│  DApp 前端开发  │ ─── 查询历史交易、用户活动
└─────────────────┘

┌─────────────────┐
│  DeFi 分析师    │ ─── 协议数据分析、链上指标
└─────────────────┘

┌─────────────────┐
│  Web3 数据产品  │ ─── 实时数据订阅、批量导出
└─────────────────┘
```

### 1.3 系统边界

**系统负责:**
- ✅ 监听和索引区块链事件
- ✅ 提供查询 API（GraphQL/REST）
- ✅ 数据聚合和统计
- ✅ 历史数据回填
- ✅ 系统监控和告警

**系统不负责:**
- ❌ 直接与智能合约交互（写操作）
- ❌ 区块链节点运维
- ❌ 前端应用开发
- ❌ 用户身份管理（仅 API Key 认证）

---

## 2. 架构设计原则

### 2.1 核心原则

| 原则 | 说明 | 体现 |
|------|------|------|
| **关注点分离** | 每个服务专注单一职责 | 索引、查询、管理分离 |
| **高内聚低耦合** | 服务间通过 gRPC 松耦合 | 明确的接口定义 |
| **最终一致性** | 容忍短暂不一致换取性能 | 异步索引 + 缓存 |
| **可测试性** | 每个模块独立可测 | 接口抽象 + Mock |
| **可观测性** | 全链路追踪和监控 | Metrics + Logs + Traces |
| **防御性编程** | 假设外部依赖不可靠 | 重试、熔断、降级 |

### 2.2 设计权衡

| 权衡点 | 选择 | 理由 |
|--------|------|------|
| **单体 vs 微服务** | 微服务 | 独立扩展、技术灵活性 |
| **同步 vs 异步** | 异步索引 | 高吞吐、解耦 |
| **强一致 vs 最终一致** | 最终一致 | 性能优先，12块确认 |
| **GraphQL vs REST** | 主 GraphQL | 灵活查询，减少 over-fetching |
| **PostgreSQL vs NoSQL** | PostgreSQL | 事务支持、JSONB 灵活性 |

---

## 3. 系统架构

### 3.1 整体架构图

```
┌────────────────────────────────────────────────────────────────┐
│                         Client Layer                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │ DApp Frontend│  │  Analytics   │  │  Admin Dashboard     │ │
│  └──────┬───────┘  └──────┬───────┘  └──────────┬───────────┘ │
└─────────┼──────────────────┼─────────────────────┼─────────────┘
          │ GraphQL/REST     │                     │ HTTP/WS
          │                  │                     │
┌─────────▼──────────────────▼─────────────────────▼─────────────┐
│                      API Gateway (8000)                         │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  - GraphQL Server (gqlgen)                               │  │
│  │  - REST API (Gin)                                        │  │
│  │  - Authentication & Rate Limiting                        │  │
│  │  - Request Logging & Metrics                             │  │
│  └──────────────────────────────────────────────────────────┘  │
└───────────────────┬──────────────────────┬──────────────────────┘
                    │ gRPC                 │ gRPC
         ┌──────────▼──────────┐  ┌────────▼─────────┐
         │  Query Service      │  │  Admin Service   │
         │      (8081)         │  │      (8082)      │
         └──────────┬──────────┘  └────────┬─────────┘
                    │ SQL/Redis            │ gRPC
                    │                      │
┌───────────────────▼──────────────────────▼─────────────────────┐
│                    Indexer Service (8080)                       │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Event Listener ──► Parser ──► Validator ──► Storage    │  │
│  │       │                                          │        │  │
│  │       └─────► Reorg Detector ◄──────────────────┘        │  │
│  └──────────────────────────────────────────────────────────┘  │
└───────────┬──────────────────────────────────────────┬─────────┘
            │ WebSocket/HTTP                           │ SQL
            │                                          │
┌───────────▼────────────┐              ┌──────────────▼──────────┐
│  Blockchain Node(s)    │              │    PostgreSQL 15        │
│  ┌──────────────────┐  │              │  ┌──────────────────┐  │
│  │ Primary RPC      │  │              │  │ events           │  │
│  │ Fallback RPC     │  │              │  │ contracts        │  │
│  │ (Alchemy/Infura) │  │              │  │ indexer_state    │  │
│  └──────────────────┘  │              │  └──────────────────┘  │
└────────────────────────┘              └─────────────────────────┘
                                                     │
                                        ┌────────────▼────────────┐
                                        │     Redis 7 Cache       │
                                        │  - Query Cache          │
                                        │  - Block State Cache    │
                                        │  - Task Queue           │
                                        └─────────────────────────┘

┌────────────────────────────────────────────────────────────────┐
│                      Monitoring Stack                           │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────────────┐│
│  │ Prometheus   │  │   Grafana    │  │  Alertmanager         ││
│  │  (Metrics)   │  │ (Dashboards) │  │  (Notifications)      ││
│  └──────────────┘  └──────────────┘  └───────────────────────┘│
└────────────────────────────────────────────────────────────────┘
```

### 3.2 数据流图

#### 3.2.1 事件索引流程

```
┌──────────────┐
│  New Block   │
│  on Chain    │
└──────┬───────┘
       │
       ▼
┌──────────────────────────────────────────┐
│  1. Indexer Service                      │
│  ┌────────────────────────────────────┐  │
│  │ Event Listener                     │  │
│  │ - Subscribe to new blocks          │  │
│  │ - Fetch block events (eth_getLogs)│  │
│  └────────────┬───────────────────────┘  │
│               ▼                          │
│  ┌────────────────────────────────────┐  │
│  │ Event Parser                       │  │
│  │ - Decode event parameters (ABI)   │  │
│  │ - Extract indexed/non-indexed args│  │
│  │ - Type conversion (BigNumber→str) │  │
│  └────────────┬───────────────────────┘  │
│               ▼                          │
│  ┌────────────────────────────────────┐  │
│  │ Reorg Detector                     │  │
│  │ - Check block hash consistency    │  │
│  │ - If reorg: rollback & re-index   │  │
│  └────────────┬───────────────────────┘  │
│               ▼                          │
│  ┌────────────────────────────────────┐  │
│  │ Batch Storage                      │  │
│  │ - Accumulate 100-500 events       │  │
│  │ - Bulk insert to PostgreSQL       │  │
│  │ - Update indexer_state             │  │
│  └────────────┬───────────────────────┘  │
└───────────────┼──────────────────────────┘
                │
                ▼
┌───────────────────────────────────────────┐
│  2. PostgreSQL                            │
│  - INSERT events (ON CONFLICT DO NOTHING) │
│  - UPDATE contracts.current_block         │
│  - Emit notification (NOTIFY)             │
└───────────────┬───────────────────────────┘
                │
                ▼
┌───────────────────────────────────────────┐
│  3. Query Service                         │
│  - Invalidate related cache               │
│  - Update aggregation cache               │
└───────────────────────────────────────────┘
```

#### 3.2.2 查询流程

```
┌──────────────┐
│  Client      │
│  GraphQL     │
│  Request     │
└──────┬───────┘
       │
       ▼
┌────────────────────────────────────┐
│  1. API Gateway                    │
│  - Authentication                  │
│  - Rate Limiting                   │
│  - Parse GraphQL Query             │
└────────┬───────────────────────────┘
         │ gRPC
         ▼
┌────────────────────────────────────┐
│  2. Query Service                  │
│  ┌──────────────────────────────┐  │
│  │ Cache Check                  │  │
│  │ - Check Redis cache          │  │
│  │ - If hit: return cached data │  │
│  └────────┬─────────────────────┘  │
│           │ Cache Miss             │
│           ▼                        │
│  ┌──────────────────────────────┐  │
│  │ Query Optimizer              │  │
│  │ - Analyze query complexity   │  │
│  │ - Select index strategy      │  │
│  └────────┬─────────────────────┘  │
└───────────┼────────────────────────┘
            │ SQL
            ▼
┌────────────────────────────────────┐
│  3. PostgreSQL                     │
│  - Execute optimized query         │
│  - Use GIN index for JSONB         │
│  - Return result set               │
└────────┬───────────────────────────┘
         │
         ▼
┌────────────────────────────────────┐
│  4. Query Service                  │
│  - Format response                 │
│  - Cache result (TTL 30s)          │
│  - Return to API Gateway           │
└────────┬───────────────────────────┘
         │ gRPC
         ▼
┌────────────────────────────────────┐
│  5. API Gateway                    │
│  - Format GraphQL response         │
│  - Log metrics                     │
│  - Return to client                │
└────────────────────────────────────┘
```

---

## 4. 微服务设计

### 4.1 服务拆分原则

基于以下维度进行服务拆分：
1. **业务能力**: 索引、查询、管理各自独立
2. **技术特性**: 不同性能要求（索引高吞吐，查询低延迟）
3. **扩展需求**: 查询服务需要水平扩展
4. **团队自治**: 不同团队可独立开发

### 4.2 服务详细设计

#### 4.2.1 Indexer Service

**职责:**
- 监听区块链新区块
- 解析智能合约事件
- 处理链重组
- 批量写入数据库

**技术特点:**
- CPU 密集（事件解析）
- 写密集（数据库操作）
- 需要保持长连接（WebSocket）

**关键组件:**

```go
// 服务结构
type IndexerService struct {
    rpcManager    *RPCManager       // RPC 节点管理
    eventListener *EventListener    // 事件监听器
    eventParser   *EventParser      // 事件解析器
    reorgDetector *ReorgDetector    // 重组检测器
    storage       *EventStorage     // 数据存储
    stateManager  *StateManager     // 状态管理
}

// 核心接口
type EventListener interface {
    Subscribe(ctx context.Context, contracts []Contract) error
    GetEvents(blockNumber uint64) ([]Event, error)
}

type EventParser interface {
    Parse(log types.Log, abi abi.ABI) (*ParsedEvent, error)
}

type ReorgDetector interface {
    CheckReorg(currentBlock *types.Block) (bool, uint64, error)
    HandleReorg(forkPoint uint64) error
}
```

**配置示例:**

```yaml
indexer:
  # RPC 配置
  rpc:
    primary: "wss://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
    fallback:
      - "https://rpc.ankr.com/eth"
      - "https://cloudflare-eth.com"
    timeout: 30s
    max_retry: 3
  
  # 索引配置
  batch_size: 100
  default_confirm_blocks: 6  # 默认使用平衡模式（6块）
  poll_interval: 6s
  max_concurrent_contracts: 5
  
  # 缓冲区配置
  event_buffer_size: 10000
  block_cache_size: 100
  
  # 确认策略预设（可在合约级别覆盖）
  confirmation_presets:
    realtime: 1   # 实时模式
    balanced: 6   # 平衡模式（推荐）
    safe: 12      # 安全模式
```

**Metrics 指标:**

```
# 索引延迟
indexer_lag_seconds

# 事件处理速率
indexer_events_processed_total

# RPC 调用统计
indexer_rpc_calls_total{endpoint, status}

# Reorg 检测
indexer_reorg_detected_total
```

---

#### 4.2.2 API Gateway

**职责:**
- 对外提供 GraphQL/REST API
- 认证和授权
- 请求限流
- 路由到后端服务

**技术特点:**
- 无状态（易于水平扩展）
- 请求转发（低 CPU 消耗）
- 需要负载均衡

**关键组件:**

```go
type APIGateway struct {
    graphqlServer *GraphQLServer
    restRouter    *gin.Engine
    
    // gRPC 客户端
    queryClient   pb.QueryServiceClient
    adminClient   pb.AdminServiceClient
    
    // 中间件
    authMiddleware      *AuthMiddleware
    rateLimitMiddleware *RateLimitMiddleware
    loggingMiddleware   *LoggingMiddleware
}

// GraphQL Resolver
type Resolver struct {
    queryClient pb.QueryServiceClient
    adminClient pb.AdminServiceClient
}
```

**认证流程:**

```
Client Request
    │
    ▼
┌─────────────────────┐
│ Extract API Key     │ ─── From Header: X-API-Key
└─────┬───────────────┘
      │
      ▼
┌─────────────────────┐
│ Validate API Key    │ ─── Redis Lookup (cached)
└─────┬───────────────┘
      │
      ├─ Valid ─────► Continue to Handler
      │
      └─ Invalid ───► Return 401 Unauthorized
```

**限流策略:**

```
Rate Limiting:
  - Per API Key: 1000 req/min (Pro), 100 req/min (Free)
  - Per IP: 10000 req/min (防止 DDoS)
  - Per Endpoint: 自定义限制

Implementation:
  - Redis + Token Bucket Algorithm
  - Sliding Window (1 minute)
```

---

#### 4.2.3 Query Service

**职责:**
- 执行数据库查询
- 查询优化
- 结果缓存
- 数据聚合

**技术特点:**
- 读密集
- 需要缓存层
- 复杂查询优化

**关键组件:**

```go
type QueryService struct {
    db           *sql.DB
    cache        *redis.Client
    queryBuilder *QueryBuilder
    aggregator   *Aggregator
}

// gRPC 服务实现
type queryServiceServer struct {
    pb.UnimplementedQueryServiceServer
    service *QueryService
}

func (s *queryServiceServer) GetEvents(
    ctx context.Context, 
    req *pb.EventQuery,
) (*pb.EventResponse, error) {
    // 1. Check cache
    cacheKey := generateCacheKey(req)
    if cached, err := s.service.cache.Get(ctx, cacheKey).Result(); err == nil {
        return unmarshalResponse(cached), nil
    }
    
    // 2. Query database
    events, err := s.service.QueryEvents(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // 3. Cache result
    s.service.cache.Set(ctx, cacheKey, marshal(events), 30*time.Second)
    
    return events, nil
}
```

**查询优化策略:**

```sql
-- 策略 1: 使用 GIN 索引 (MVP)
CREATE INDEX idx_events_args_gin ON events USING GIN(args);

SELECT * FROM events 
WHERE args @> '{"from": "0x123..."}' 
  AND block_number > 1000000
ORDER BY block_number DESC
LIMIT 100;

-- 策略 2: 专用地址表 (优化阶段)
CREATE TABLE event_addresses (
    event_id BIGINT,
    param_name VARCHAR(50),
    address VARCHAR(42),
    INDEX idx_address (address, param_name)
);

SELECT e.* FROM events e
JOIN event_addresses ea ON e.id = ea.event_id
WHERE ea.address = '0x123...'
  AND ea.param_name IN ('from', 'to')
ORDER BY e.block_number DESC
LIMIT 100;
```

**缓存策略:**

```
┌─────────────────────────────────────┐
│  Cache Key Design                   │
├─────────────────────────────────────┤
│  Format: query:{hash}:{version}     │
│                                     │
│  Examples:                          │
│  - query:abc123:v1                  │
│  - stats:0x456...:v1                │
│  - address:0x789...:page1:v1        │
└─────────────────────────────────────┘

TTL Strategy:
  - Hot queries: 30s
  - Stats: 5min
  - Historical data: 1hour

Invalidation:
  - On new events: Invalidate related queries
  - Manual: Admin API trigger
```

---

#### 4.2.4 Admin Service

**职责:**
- 合约管理
- 历史数据回填
- 系统配置
- 监控和告警

**关键功能:**

```go
type AdminService struct {
    db              *sql.DB
    indexerClient   pb.IndexerServiceClient
    backfillManager *BackfillManager
    alertManager    *AlertManager
}

// 历史数据回填
type BackfillManager struct {
    taskQueue   *redis.Client
    workers     []*BackfillWorker
}

type BackfillTask struct {
    ContractAddress string
    FromBlock       uint64
    ToBlock         uint64
    ChunkSize       uint64
    Status          string // pending, running, completed, failed
    Progress        float64
}

func (m *BackfillManager) StartBackfill(
    ctx context.Context,
    task *BackfillTask,
) error {
    // 1. 分片任务
    chunks := splitIntoChunks(task.FromBlock, task.ToBlock, task.ChunkSize)
    
    // 2. 推送到队列
    for _, chunk := range chunks {
        m.taskQueue.LPush(ctx, "backfill:queue", marshal(chunk))
    }
    
    // 3. Worker 消费
    // Workers 从队列中获取任务并执行
    
    return nil
}
```

---

### 4.3 服务间通信

#### 4.3.1 gRPC 接口定义

```protobuf
// query_service.proto
syntax = "proto3";

service QueryService {
  rpc GetEvents(EventQuery) returns (EventResponse);
  rpc GetEventsByAddress(AddressQuery) returns (EventResponse);
  rpc GetContractStats(StatsQuery) returns (StatsResponse);
}

message EventQuery {
  string contract_address = 1;
  string event_name = 2;
  uint64 from_block = 3;
  uint64 to_block = 4;
  int32 limit = 5;
  string cursor = 6;
}

message EventResponse {
  repeated Event events = 1;
  PageInfo page_info = 2;
  int32 total_count = 3;
}

message Event {
  string id = 1;
  string contract_address = 2;
  string event_name = 3;
  uint64 block_number = 4;
  int64 block_timestamp = 5;
  string transaction_hash = 6;
  map<string, string> args = 7;
}
```

#### 4.3.2 通信模式

| 场景 | 模式 | 说明 |
|------|------|------|
| API Gateway → Query Service | 同步 gRPC | 请求-响应 |
| API Gateway → Admin Service | 同步 gRPC | 请求-响应 |
| Indexer → Database | 同步 SQL | 批量写入 |
| Admin → Indexer | 异步队列 | 回填任务 |

---

## 5. 数据架构

### 5.1 数据模型

#### 5.1.1 ER 图

```
┌──────────────────────────────────┐
│         contracts                 │
├──────────────────────────────────┤
│ id (PK)                          │
│ address (UNIQUE)                 │
│ name                             │
│ abi (JSONB)                      │
│ start_block                      │
│ current_block                    │
│ confirm_blocks (默认6)           │
│ is_active                        │
│ created_at                       │
│ updated_at                       │
└───────────┬──────────────────────┘
            │ 1
            │
            │ N
            ▼
┌──────────────────────────────────┐
│         events                    │
├──────────────────────────────────┤
│ id (PK)                          │
│ contract_id (FK)                 │
│ contract_address                 │
│ event_name                       │
│ block_number                     │
│ block_timestamp                  │
│ transaction_hash                 │
│ transaction_index                │
│ log_index                        │
│ args (JSONB)                     │
│ raw_log (JSONB)                  │
│ created_at                       │
│ UNIQUE(tx_hash, log_index)       │
└───────────┬──────────────────────┘
            │ 1
            │
            │ N (可选，优化阶段)
            ▼
┌──────────────────────────────────┐
│      event_addresses              │
├──────────────────────────────────┤
│ id (PK)                          │
│ event_id (FK)                    │
│ param_name                       │
│ address                          │
│ INDEX(address, param_name)       │
└──────────────────────────────────┘

┌──────────────────────────────────┐
│      indexer_state                │
├──────────────────────────────────┤
│ contract_id (PK, FK)             │
│ last_indexed_block               │
│ last_indexed_at                  │
│ is_syncing                       │
│ error_message                    │
└──────────────────────────────────┘
```

#### 5.1.2 索引策略

```sql
-- 主表索引
CREATE INDEX idx_events_contract 
    ON events(contract_address, event_name);

CREATE INDEX idx_events_block 
    ON events(block_number DESC);

CREATE INDEX idx_events_timestamp 
    ON events(block_timestamp DESC);

CREATE INDEX idx_events_tx 
    ON events(transaction_hash);

-- JSONB 索引 (MVP)
CREATE INDEX idx_events_args_gin 
    ON events USING GIN(args);

-- 复合索引 (优化)
CREATE INDEX idx_events_contract_block 
    ON events(contract_address, block_number DESC);

-- 分区表 (大数据量)
-- 按月分区
CREATE TABLE events_2025_01 PARTITION OF events
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
```

### 5.2 数据分片策略

#### 5.2.1 垂直分片

```
┌────────────────────────────────┐
│  events_hot (近 3 个月)         │ ─── SSD, 频繁访问
├────────────────────────────────┤
│  events_warm (3-12 个月)        │ ─── SSD, 偶尔访问
├────────────────────────────────┤
│  events_cold (12 个月+)         │ ─── HDD/S3, 归档
└────────────────────────────────┘
```

#### 5.2.2 水平分片

```
# 按合约地址分片 (未来扩展)
Shard 1: contracts starting with 0x0-0x7
Shard 2: contracts starting with 0x8-0xF

# 路由逻辑
shard_id = hash(contract_address) % shard_count
```

### 5.3 数据一致性

#### 5.3.1 幂等性保证

```sql
-- 使用 UNIQUE 约束
INSERT INTO events (...) 
VALUES (...)
ON CONFLICT (transaction_hash, log_index) 
DO NOTHING;

-- 或使用 UPSERT
INSERT INTO events (...) 
VALUES (...)
ON CONFLICT (transaction_hash, log_index) 
DO UPDATE SET updated_at = NOW();
```

#### 5.3.2 事务隔离

```go
func (s *EventStorage) BatchInsertEvents(
    ctx context.Context, 
    events []*Event,
) error {
    tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
        Isolation: sql.LevelReadCommitted,
    })
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // 批量插入
    _, err = tx.ExecContext(ctx, insertSQL, args...)
    if err != nil {
        return err
    }
    
    // 更新状态
    _, err = tx.ExecContext(ctx, updateStateSQL, stateArgs...)
    if err != nil {
        return err
    }
    
    return tx.Commit()
}
```

---

## 6. API 设计

### 6.1 GraphQL Schema

```graphql
# 标量类型
scalar DateTime
scalar BigInt
scalar Address

# 核心类型
type Event {
  id: ID!
  contractAddress: Address!
  eventName: String!
  blockNumber: BigInt!
  blockTimestamp: DateTime!
  transactionHash: String!
  transactionIndex: Int!
  logIndex: Int!
  args: [EventArg!]!
}

type EventArg {
  key: String!
  value: String!
  type: String!
}

# 连接类型 (Relay Cursor Pagination)
type EventConnection {
  edges: [EventEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type EventEdge {
  node: Event!
  cursor: String!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

# 输入类型
input EventFilter {
  contractAddress: Address
  eventName: String
  fromBlock: BigInt
  toBlock: BigInt
  addresses: [Address!]  # 查询参数中包含的地址
}

input PaginationInput {
  first: Int
  after: String
  last: Int
  before: String
}

# 查询
type Query {
  # 基础查询
  events(
    filter: EventFilter
    pagination: PaginationInput
  ): EventConnection!
  
  # 按交易查询
  eventsByTransaction(txHash: String!): [Event!]!
  
  # 按地址查询
  eventsByAddress(
    address: Address!
    pagination: PaginationInput
  ): EventConnection!
  
  # 合约信息
  contract(address: Address!): Contract
  contracts(isActive: Boolean): [Contract!]!
  
  # 统计
  contractStats(address: Address!): ContractStats!
}

# 变更
type Mutation {
  # 添加合约
  addContract(input: AddContractInput!): AddContractPayload!
  
  # 删除合约
  removeContract(address: Address!): RemoveContractPayload!
  
  # 触发回填
  triggerBackfill(
    address: Address!
    fromBlock: BigInt!
    toBlock: BigInt!
  ): BackfillPayload!
}

# 订阅 (可选)
type Subscription {
  # 新事件订阅
  newEvents(contractAddress: Address): Event!
}
```

### 6.2 REST API

```
# 事件查询
GET    /api/v1/events
       ?contract=0x...
       &event_name=Transfer
       &from_block=1000000
       &to_block=1001000
       &limit=50
       &cursor=abc123

# 按交易查询
GET    /api/v1/events/tx/:txHash

# 按地址查询
GET    /api/v1/events/address/:address
       ?limit=50
       &cursor=xyz789

# 合约管理
GET    /api/v1/contracts
POST   /api/v1/contracts
DELETE /api/v1/contracts/:address

# 统计
GET    /api/v1/contracts/:address/stats

# 健康检查
GET    /api/v1/health
GET    /api/v1/health/indexer
```

### 6.3 错误处理

```graphql
# GraphQL 错误格式
{
  "errors": [
    {
      "message": "Contract not found",
      "extensions": {
        "code": "CONTRACT_NOT_FOUND",
        "address": "0x123..."
      }
    }
  ]
}

# REST 错误格式
{
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Invalid block number",
    "details": {
      "parameter": "from_block",
      "value": "-1"
    }
  }
}
```

---

## 7. 技术栈选型

### 7.1 技术决策矩阵

| 组件 | 选择 | 备选方案 | 选择理由 |
|------|------|----------|----------|
| **编程语言** | Go 1.21+ | Rust, TypeScript | 高性能、并发支持、生态完善 |
| **Web3 库** | go-ethereum | ethers-go | 官方库、功能完整 |
| **GraphQL** | gqlgen | graphql-go | 代码生成、类型安全 |
| **HTTP 框架** | Gin | Fiber, Echo | 性能优秀、中间件丰富 |
| **gRPC** | grpc-go | - | 服务间通信标准 |
| **数据库** | PostgreSQL 15 | MySQL, MongoDB | JSONB、事务、性能 |
| **缓存** | Redis 7 | Memcached | 数据结构丰富、持久化 |
| **消息队列** | Redis Streams | RabbitMQ, Kafka | 简单、与 Redis 集成 |
| **监控** | Prometheus | Datadog | 开源、Kubernetes 友好 |
| **日志** | Zap | Logrus | 高性能结构化日志 |
| **容器** | Docker | - | 标准化部署 |
| **编排** | Kubernetes | Docker Swarm | 生态完善、生产级 |

### 7.2 依赖管理

```go
// go.mod
module github.com/yourorg/event-indexer

go 1.21

require (
    github.com/ethereum/go-ethereum v1.13.0
    github.com/99designs/gqlgen v0.17.40
    github.com/gin-gonic/gin v1.9.1
    google.golang.org/grpc v1.59.0
    github.com/lib/pq v1.10.9
    github.com/redis/go-redis/v9 v9.3.0
    github.com/prometheus/client_golang v1.17.0
    go.uber.org/zap v1.26.0
)
```

---

## 8. 关键设计决策

### ADR-001: 选择微服务架构而非单体

**背景:**
需要决定系统架构模式。

**决策:**
采用微服务架构，拆分为 4 个独立服务。

**理由:**
1. **独立扩展**: 查询服务可独立水平扩展
2. **技术灵活**: 不同服务可用不同技术栈
3. **故障隔离**: 索引服务故障不影响查询
4. **团队自治**: 适合多人协作开发

**后果:**
- ✅ 更好的可扩展性和可维护性
- ⚠️ 增加运维复杂度
- ⚠️ 需要服务发现和监控

**状态:** ✅ Accepted

---

### ADR-002: 使用 GraphQL 作为主要 API

**背景:**
需要选择 API 设计风格。

**决策:**
主要使用 GraphQL，辅助提供 REST API。

**理由:**
1. **灵活查询**: 客户端可自定义返回字段
2. **减少请求**: 一次请求获取所有需要的数据
3. **类型安全**: Schema 提供强类型定义
4. **自文档化**: Playground 提供交互式文档

**后果:**
- ✅ 更好的开发体验
- ✅ 减少 over-fetching 和 under-fetching
- ⚠️ 学习曲线较陡

**状态:** ✅ Accepted

---

### ADR-003: PostgreSQL JSONB vs 专用地址表

**背景:**
`eventsByAddress` 查询需要高性能。

**决策:**
MVP 阶段使用 GIN 索引，必要时引入专用地址表。

**理由:**
1. **渐进优化**: 先验证功能，再优化性能
2. **灵活性**: JSONB 适合动态 schema
3. **成本**: 专用表增加 50% 存储和写入成本

**实施路径:**
- Phase 1-2: GIN 索引
- Phase 3: 如果 P95 > 500ms，引入地址表

**状态:** ✅ Accepted

---

### ADR-004: 可配置确认块策略

**背景:**
需要在速度和数据准确性间平衡。不同应用场景对延迟和安全性的要求不同。

**决策:**
实现可配置的确认块策略，允许每个合约选择不同的确认级别。

**三种预设策略:**

| 策略 | 确认块数 | 延迟 | 准确率 | 适用场景 |
|------|---------|------|--------|---------|
| **实时模式** | 1 块 | ~12 秒 | ~99% | Demo、游戏、实时通知 |
| **平衡模式** (默认) | 6 块 | ~72 秒 | ~99.99% | 大多数生产应用 |
| **安全模式** | 12 块 | ~144 秒 | ~99.9999% | 金融、支付、审计 |

**实现细节:**
```go
type Contract struct {
    ConfirmBlocks int32 // 1, 6, or 12
}

// 索引器在检查时
if currentBlock - eventBlock >= contract.ConfirmBlocks {
    // 认为事件已确认，可以索引
}
```

**理由:**
1. **灵活性**: 不同应用有不同需求
2. **风险控制**: 用户明确选择速度vs安全的权衡
3. **最佳实践**: 参考 Alchemy/Infura 等主流服务
4. **可观测性**: 可监控不同策略的实际表现

**权衡:**
- ✅ 满足不同场景需求
- ✅ 默认6块是最佳平衡点
- ⚠️ 增加配置复杂度（通过合理默认值缓解）
- ⚠️ 需要文档说明权衡

**状态:** ✅ Accepted

---

## 9. 部署架构

### 9.1 本地开发环境

```yaml
# docker-compose.yml
version: '3.9'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: indexer
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
  
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
  
  ganache:
    image: trufflesuite/ganache:latest
    ports:
      - "8545:8545"
    command: --deterministic --accounts 10
  
  indexer:
    build: ./services/indexer-service
    environment:
      - DATABASE_URL=postgres://postgres:postgres@postgres:5432/indexer
      - REDIS_URL=redis://redis:6379
      - RPC_URL=http://ganache:8545
    depends_on:
      - postgres
      - redis
      - ganache
  
  api-gateway:
    build: ./services/api-gateway
    ports:
      - "8000:8000"
    depends_on:
      - indexer
      - query-service
  
  query-service:
    build: ./services/query-service
    depends_on:
      - postgres
      - redis
  
  admin-service:
    build: ./services/admin-service
    ports:
      - "8082:8082"
```

### 9.2 Kubernetes 生产环境

```yaml
# k8s/indexer-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: indexer-service
spec:
  replicas: 2
  selector:
    matchLabels:
      app: indexer-service
  template:
    metadata:
      labels:
        app: indexer-service
    spec:
      containers:
      - name: indexer
        image: yourregistry/indexer-service:latest
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: url
        - name: RPC_URL
          valueFrom:
            configMapKeyRef:
              name: app-config
              key: rpc_url
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
# HorizontalPodAutoscaler
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-gateway-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-gateway
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### 9.3 免费/低成本云部署方案

#### 方案 A: Railway.app (推荐用于 Portfolio)

**总成本**: $5-10/月 (Railway 提供 $5 免费额度)

```
┌─────────────────────────────────────────────────────────┐
│                    Railway.app                           │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │         Railway Load Balancer (自动)               │ │
│  └────────────┬───────────────────────────────────────┘ │
│               │                                          │
│  ┌────────────▼───────────────────────────────────────┐ │
│  │  Services (Docker Containers)                      │ │
│  │                                                    │ │
│  │  ┌─────────────┐  ┌──────────────┐                │ │
│  │  │ API Gateway │  │ Query Service│                │ │
│  │  │   ($2-3)    │  │   ($1-2)     │                │ │
│  │  └─────────────┘  └──────────────┘                │ │
│  │                                                    │ │
│  │  ┌─────────────┐  ┌──────────────┐                │ │
│  │  │  Indexer    │  │ Admin Service│                │ │
│  │  │   ($2-3)    │  │   ($1)       │                │ │
│  │  └─────────────┘  └──────────────┘                │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
│  ┌─────────────────┐        ┌──────────────────────┐   │
│  │   PostgreSQL    │        │       Redis          │   │
│  │  (Plugin $2)    │        │    (Plugin $1)       │   │
│  └─────────────────┘        └──────────────────────┘   │
└─────────────────────────────────────────────────────────┘

# 外部免费服务
┌─────────────────┐        ┌──────────────────────┐
│  Alchemy/Infura │        │   BetterUptime       │
│  (RPC Free Tier)│        │ (Monitoring Free)    │
└─────────────────┘        └──────────────────────┘
```

**优点**:
- ✅ GitHub 直接部署，支持 Docker
- ✅ 自动 HTTPS/SSL
- ✅ 简单易用，适合快速原型
- ✅ $5/月免费额度足够小规模运行

**缺点**:
- ⚠️ 免费额度有限，需要监控用量
- ⚠️ 性能不如专业云平台

---

#### 方案 B: Render.com (完全免费，有限制)

**总成本**: $0 (但服务会 sleep)

```
┌─────────────────────────────────────────────────────────┐
│                      Render.com                          │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │  Web Services (Free Tier - Sleep after 15min)     │ │
│  │                                                    │ │
│  │  ┌─────────────┐  ┌──────────────┐                │ │
│  │  │ API Gateway │  │ Query Service│                │ │
│  │  │   (Free)    │  │   (Free)     │                │ │
│  │  └─────────────┘  └──────────────┘                │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
│  ⚠️  Background Workers (需付费 $7/月)                  │
│  ┌────────────────────────────────────────────────────┐ │
│  │  ┌─────────────┐  ┌──────────────┐                │ │
│  │  │  Indexer    │  │ Admin Service│                │ │
│  │  │  ($7/mo)    │  │   (可选)     │                │ │
│  │  └─────────────┘  └──────────────┘                │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘

# 外部服务
┌─────────────────┐        ┌──────────────────────┐
│  Supabase       │        │    Upstash Redis     │
│ (PostgreSQL)    │        │    (Serverless)      │
│  250MB Free     │        │    10K cmd/day       │
└─────────────────┘        └──────────────────────┘
```

**优点**:
- ✅ 完全免费（Web Services）
- ✅ PostgreSQL 免费 90 天
- ✅ 自动 HTTPS/SSL

**缺点**:
- ⚠️ 免费层服务 15 分钟无活动会 sleep（冷启动 ~30秒）
- ⚠️ Indexer 必须付费（需要持续运行）

---

#### 方案 C: 混合架构 (最优成本)

**总成本**: ~$5/月

```
┌──────────────────────────────────────────────────────┐
│               混合部署架构                             │
└──────────────────────────────────────────────────────┘

前端/API (如果有)
┌─────────────────┐
│   Vercel        │ ─── 免费，适合 Next.js/React
│  (API Gateway)  │      也可托管轻量 API Routes
└────────┬────────┘
         │ HTTPS
         ▼
┌─────────────────────────────────────────────────────┐
│              Railway.app ($5)                        │
│                                                     │
│  ┌─────────────┐  ┌──────────────┐                 │
│  │  Indexer    │  │ Query Service│                 │
│  │  (必须持续) │  │   (按需)     │                 │
│  └─────────────┘  └──────────────┘                 │
└─────────────────────────────────────────────────────┘

数据层
┌─────────────────┐        ┌──────────────────────┐
│  Supabase       │        │   Upstash Redis      │
│ (PostgreSQL)    │        │   (Serverless)       │
│   免费 500MB    │        │   免费 10K/day       │
└─────────────────┘        └──────────────────────┘

RPC 节点
┌─────────────────┐        ┌──────────────────────┐
│  Alchemy        │        │   Infura             │
│  (Primary)      │        │   (Fallback)         │
│  免费 300M CU   │        │   免费 100K/day      │
└─────────────────┘        └──────────────────────┘

监控
┌─────────────────┐
│  BetterUptime   │ ─── 免费监控和告警
└─────────────────┘
```

**优点**:
- ✅ 成本最低（~$5/月）
- ✅ 利用各平台免费额度
- ✅ 性能足够 Portfolio 展示

---

#### 方案 D: Oracle Cloud Free Tier (永久免费)

**总成本**: $0 (永久)

```
┌─────────────────────────────────────────────────────────┐
│           Oracle Cloud Infrastructure (OCI)              │
│                   Always Free Tier                       │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │  Compute Instances (Always Free)                   │ │
│  │                                                    │ │
│  │  VM.Standard.A1.Flex (Ampere ARM)                 │ │
│  │  - 4 OCPUs (ARM64)                                │ │
│  │  - 24 GB RAM                                      │ │
│  │  - 可拆分为多个小实例                              │ │
│  │                                                    │ │
│  │  部署方式: Docker Compose / K3s                    │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
│  ┌─────────────────┐        ┌──────────────────────┐   │
│  │  Block Volume   │        │   Object Storage     │   │
│  │  200GB SSD      │        │   10GB               │   │
│  └─────────────────┘        └──────────────────────┘   │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │  Flexible Load Balancer (Always Free)             │ │
│  │  - 10 Mbps bandwidth                              │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

**优点**:
- ✅ 完全免费，永久
- ✅ 资源充足（24GB RAM 足够运行所有服务）
- ✅ 完全控制

**缺点**:
- ⚠️ ARM 架构（需要构建 ARM64 镜像）
- ⚠️ 配置较复杂
- ⚠️ 网络速度较慢（但够用）

---

### 9.4 推荐部署策略

#### 🎯 Portfolio 展示项目（推荐）

```yaml
选择: Railway.app ($5/月) + 外部免费服务

部署清单:
  ✅ Railway Services:
     - indexer-service (主要，必须运行)
     - api-gateway (按需)
     - query-service (可选，合并到 api-gateway)
  
  ✅ 外部服务:
     - Database: Supabase (PostgreSQL 免费 500MB)
     - Cache: Upstash Redis (免费 10K 命令/天)
     - RPC: Alchemy (免费 300M 计算单元/月)
     - Monitoring: BetterUptime (免费)
  
  预期成本: $0-5/月
  足够支撑: 5-10 个合约，1000 events/天
```

#### 💰 零成本方案（有限制）

```yaml
选择: Oracle Cloud Free Tier

部署方式:
  1. 申请 Oracle Cloud 账号
  2. 创建 VM.Standard.A1.Flex 实例 (4 OCPU, 24GB RAM)
  3. 使用 Docker Compose 部署所有服务
  4. 配置 Nginx 反向代理
  5. 使用 Let's Encrypt 免费 SSL

优化建议:
  - 使用 Docker 资源限制避免 OOM
  - 配置 swap 空间
  - 定期清理日志和临时文件
  
  示例配置:
  services:
    indexer:
      mem_limit: 1g
      mem_reservation: 512m
    api-gateway:
      mem_limit: 512m
      mem_reservation: 256m
```

#### 🚀 生产级方案（低成本）

```yaml
选择: Fly.io + Supabase

部署配置:
  ✅ Fly.io:
     - 3x Shared CPU (1x 256MB free, 2x $1.94/mo)
     - 3GB 持久化存储
     - 自动全球部署
  
  ✅ Supabase:
     - PostgreSQL Pro ($25/月)
     - 8GB 数据库，无连接限制
     - 自动备份
  
  总成本: ~$30/月
  适合: 生产环境，多用户
```

---

### 9.5 各方案对比

| 平台 | 月成本 | 部署难度 | 适用场景 | 限制 |
|------|--------|----------|----------|------|
| **Railway** | $5 | ⭐⭐ | Portfolio | 免费额度有限 |
| **Render** | $0-7 | ⭐ | Demo | Free tier 会 sleep |
| **Oracle Cloud** | $0 | ⭐⭐⭐⭐ | 长期运行 | 需要运维经验 |
| **Fly.io** | $0-10 | ⭐⭐⭐ | 小规模生产 | 配置复杂 |
| **混合方案** | $5 | ⭐⭐⭐ | 最优成本 | 管理多平台 |
| **AWS/GCP** | $50+ | ⭐⭐⭐⭐⭐ | 企业级 | 成本高 |

---

### 9.6 一键部署模板

#### Railway 部署配置

```yaml
# railway.yaml
services:
  indexer:
    build:
      dockerfile: services/indexer-service/Dockerfile
    environment:
      DATABASE_URL: ${{Postgres.DATABASE_URL}}
      REDIS_URL: ${{Redis.REDIS_URL}}
      RPC_URL: ${{RPC_URL}}
    healthcheck:
      path: /health
      interval: 30s
    restart: always
  
  api-gateway:
    build:
      dockerfile: services/api-gateway/Dockerfile
    environment:
      DATABASE_URL: ${{Postgres.DATABASE_URL}}
      REDIS_URL: ${{Redis.REDIS_URL}}
      INDEXER_GRPC: indexer:8080
    healthcheck:
      path: /health
      interval: 30s
    domains:
      - myindexer.railway.app

plugins:
  postgres:
    plan: starter  # $5/month
  redis:
    plan: starter  # $3/month
```

#### Docker Compose (Oracle Cloud / VPS)

```yaml
# docker-compose.prod.yml
version: '3.9'

services:
  indexer:
    image: ghcr.io/yourorg/indexer-service:latest
    restart: unless-stopped
    environment:
      DATABASE_URL: ${DATABASE_URL}
      REDIS_URL: redis://redis:6379
      RPC_URL: ${RPC_URL}
    mem_limit: 1g
    cpus: 1.0
  
  api-gateway:
    image: ghcr.io/yourorg/api-gateway:latest
    restart: unless-stopped
    ports:
      - "8000:8000"
    environment:
      DATABASE_URL: ${DATABASE_URL}
      REDIS_URL: redis://redis:6379
    mem_limit: 512m
    cpus: 0.5
  
  redis:
    image: redis:7-alpine
    restart: unless-stopped
    volumes:
      - redis_data:/data
    mem_limit: 256m
  
  postgres:
    image: postgres:15-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: indexer
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    mem_limit: 2g

volumes:
  postgres_data:
  redis_data:
```

---

### 9.7 成本优化建议

```yaml
降低成本的技巧:

1. RPC 节点优化:
   ✅ 使用免费 RPC（Alchemy 300M CU/月）
   ✅ 实现请求缓存（减少重复调用）
   ✅ 批量获取事件（减少 API 调用次数）
   
   示例: 
   - 单独获取: 1000 blocks = 1000 requests
   - 批量获取: 1000 blocks = 10 requests (100/batch)
   💰 节省 99% RPC 费用

2. 数据库优化:
   ✅ 使用 Supabase 免费层（500MB 足够小项目）
   ✅ 定期清理旧数据（保留 3-6 个月）
   ✅ 使用数据库连接池（减少连接数）
   
   免费额度足够:
   - 500MB ≈ 50 万 events
   - 1GB ≈ 100 万 events

3. 缓存优化:
   ✅ Upstash Redis 免费 10K 命令/天
   ✅ 使用内存缓存（减少 Redis 调用）
   ✅ 调整 TTL 平衡命中率和新鲜度
   
   10K/天 = 每 8.6 秒 1 次命令（足够用）

4. 服务合并:
   如果流量不大，可以合并服务:
   ✅ Query Service → API Gateway
   ✅ Admin Service → API Gateway
   
   💰 节省 50% 计算资源

5. 冷启动优化:
   如果使用 Render 免费层:
   ✅ 设置定时 ping（保持唤醒）
   ✅ 使用 UptimeRobot（免费监控）
   ✅ 接受 30 秒冷启动（Portfolio 可接受）
```

---

## 10. 安全设计

### 10.1 认证与授权

```
┌─────────────────────────────────────┐
│  Authentication Flow                │
├─────────────────────────────────────┤
│  1. Client includes API Key         │
│     Header: X-API-Key: xxx...       │
│                                     │
│  2. API Gateway validates           │
│     - Check Redis cache             │
│     - If miss, query DB             │
│     - Verify key active + not       │
│       expired                       │
│                                     │
│  3. Attach metadata to request      │
│     - User ID                       │
│     - Rate limit tier               │
│     - Permissions                   │
│                                     │
│  4. Forward to backend service      │
└─────────────────────────────────────┘
```

### 10.2 数据安全

```yaml
# 敏感数据保护
sensitive_data:
  - api_keys: 使用 bcrypt 哈希存储
  - rpc_keys: 使用 Kubernetes Secrets
  - db_passwords: 使用 AWS Secrets Manager

# 传输加密
transport:
  - external: TLS 1.3 (HTTPS)
  - internal: mTLS (gRPC)

# 数据备份
backup:
  - PostgreSQL: 每日全量 + 每小时增量
  - Redis: RDB + AOF
  - 保留周期: 30 天
```

### 10.3 攻击防护

| 攻击类型 | 防护措施 |
|---------|---------|
| **DDoS** | CloudFront + WAF, Rate Limiting |
| **SQL Injection** | Prepared Statements, ORM |
| **API Abuse** | Rate Limiting, API Key |
| **Data Leak** | 最小权限原则, 审计日志 |
| **MITM** | TLS, Certificate Pinning |

---

## 11. 性能优化

### 11.1 数据库优化

```sql
-- 连接池配置
max_connections = 100
shared_buffers = 4GB
effective_cache_size = 12GB
maintenance_work_mem = 1GB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200

-- 查询优化
-- 1. 使用 EXPLAIN ANALYZE
EXPLAIN ANALYZE
SELECT * FROM events
WHERE contract_address = '0x...'
  AND block_number > 1000000;

-- 2. 避免全表扫描
CREATE INDEX CONCURRENTLY idx_events_contract_block
ON events(contract_address, block_number DESC);

-- 3. 分区表
CREATE TABLE events (
    id BIGSERIAL NOT NULL,
    block_timestamp TIMESTAMP NOT NULL,
    ...
) PARTITION BY RANGE (block_timestamp);
```

### 11.2 缓存优化

```go
// 多层缓存
type CacheManager struct {
    l1 *LocalCache  // 内存缓存 (10MB)
    l2 *redis.Client // Redis 缓存
}

func (c *CacheManager) Get(key string) (interface{}, error) {
    // L1: 内存缓存 (纳秒级)
    if val, ok := c.l1.Get(key); ok {
        return val, nil
    }
    
    // L2: Redis 缓存 (毫秒级)
    val, err := c.l2.Get(context.Background(), key).Result()
    if err == nil {
        c.l1.Set(key, val, 30*time.Second)
        return val, nil
    }
    
    // L3: 数据库 (10-100ms)
    return nil, ErrCacheMiss
}
```

### 11.3 并发优化

```go
// Goroutine 池
type WorkerPool struct {
    workers   int
    taskQueue chan Task
}

func (p *WorkerPool) Start() {
    for i := 0; i < p.workers; i++ {
        go func() {
            for task := range p.taskQueue {
                task.Execute()
            }
        }()
    }
}

// 批量处理
func (s *IndexerService) ProcessBlocks(blocks []*types.Block) error {
    var wg sync.WaitGroup
    semaphore := make(chan struct{}, 10) // 限制并发数
    
    for _, block := range blocks {
        wg.Add(1)
        semaphore <- struct{}{}
        
        go func(b *types.Block) {
            defer wg.Done()
            defer func() { <-semaphore }()
            
            s.processBlock(b)
        }(block)
    }
    
    wg.Wait()
    return nil
}
```

---

## 12. 可观测性

### 12.1 监控指标

```go
// Prometheus Metrics
var (
    // 索引器指标
    indexerLag = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "indexer_lag_seconds",
        Help: "Seconds behind the chain head",
    })
    
    eventsProcessed = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "indexer_events_processed_total",
            Help: "Total events processed",
        },
        []string{"contract", "event_name"},
    )
    
    // API 指标
    apiRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "api_request_duration_seconds",
            Help: "API request duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint", "status"},
    )
    
    // 数据库指标
    dbQueryDuration = promauto.NewHistogram(
        prometheus.HistogramOpts{
            Name: "db_query_duration_seconds",
            Help: "Database query duration",
        },
    )
)
```

### 12.2 日志规范

```go
// 结构化日志
logger.Info("Event indexed",
    zap.String("contract", event.ContractAddress),
    zap.String("event_name", event.EventName),
    zap.Uint64("block_number", event.BlockNumber),
    zap.String("tx_hash", event.TxHash),
    zap.Duration("duration", time.Since(start)),
)

// 错误日志
logger.Error("Failed to fetch events",
    zap.Error(err),
    zap.String("rpc_endpoint", rpc.URL),
    zap.Uint64("block_number", blockNumber),
    zap.Int("retry_count", retryCount),
)
```

### 12.3 分布式追踪

```go
// OpenTelemetry 追踪
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func (s *IndexerService) IndexBlock(ctx context.Context, blockNum uint64) error {
    ctx, span := otel.Tracer("indexer").Start(ctx, "IndexBlock")
    defer span.End()
    
    span.SetAttributes(
        attribute.Int64("block_number", int64(blockNum)),
    )
    
    // 子 span
    events, err := s.fetchEvents(ctx, blockNum)
    if err != nil {
        span.RecordError(err)
        return err
    }
    
    return s.storeEvents(ctx, events)
}
```

---

## 13. 容错与高可用

### 13.1 故障模式分析

| 故障模式 | 影响 | 检测 | 恢复 |
|---------|------|------|------|
| **RPC 节点失败** | 索引暂停 | 健康检查 | Fallback 节点 |
| **数据库主节点故障** | 写入失败 | 连接超时 | 切换到从节点 |
| **Redis 故障** | 缓存失效 | 连接错误 | 直接查数据库 |
| **API Gateway 故障** | 查询不可用 | 负载均衡检测 | 流量切换 |
| **索引器故障** | 索引延迟增加 | 延迟监控 | 自动重启 |

### 13.2 重试策略

```go
// 指数退避重试
func RetryWithBackoff(
    ctx context.Context,
    fn func() error,
    maxRetries int,
) error {
    backoff := time.Second
    
    for i := 0; i < maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }
        
        if !isRetryable(err) {
            return err
        }
        
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(backoff):
            backoff *= 2
            if backoff > 30*time.Second {
                backoff = 30 * time.Second
            }
        }
    }
    
    return errors.New("max retries exceeded")
}
```

### 13.3 熔断器

```go
type CircuitBreaker struct {
    maxFailures  int
    resetTimeout time.Duration
    state        State
    failures     int
    lastFailure  time.Time
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    if cb.state == StateOpen {
        if time.Since(cb.lastFailure) > cb.resetTimeout {
            cb.state = StateHalfOpen
        } else {
            return ErrCircuitOpen
        }
    }
    
    err := fn()
    if err != nil {
        cb.onFailure()
        return err
    }
    
    cb.onSuccess()
    return nil
}
```

---

## 14. 扩展性设计

### 14.1 水平扩展

```
# Query Service 水平扩展
┌─────────────┐
│ Load        │
│ Balancer    │
└──────┬──────┘
       │
   ┌───┴────┐
   │        │
   ▼        ▼
┌─────┐  ┌─────┐  ┌─────┐
│ QS1 │  │ QS2 │  │ QSN │ ─── 无状态,易扩展
└─────┘  └─────┘  └─────┘
   │        │        │
   └────────┼────────┘
            ▼
    ┌───────────────┐
    │  PostgreSQL   │
    │  (Read Pool)  │
    └───────────────┘
```

### 14.2 垂直扩展

```yaml
# 资源配置
services:
  indexer:
    small:   { cpu: 1, memory: 2Gi }
    medium:  { cpu: 2, memory: 4Gi }
    large:   { cpu: 4, memory: 8Gi }
  
  query:
    small:   { cpu: 0.5, memory: 1Gi }
    medium:  { cpu: 1, memory: 2Gi }
    large:   { cpu: 2, memory: 4Gi }
```

### 14.3 多链支持

```go
// 链配置
type ChainConfig struct {
    ChainID       int64
    Name          string
    RPCURL        string
    BlockTime     time.Duration
    ConfirmBlocks int
}

var supportedChains = map[string]ChainConfig{
    "ethereum": {
        ChainID:       1,
        BlockTime:     12 * time.Second,
        ConfirmBlocks: 12,
    },
    "polygon": {
        ChainID:       137,
        BlockTime:     2 * time.Second,
        ConfirmBlocks: 128,
    },
    "bsc": {
        ChainID:       56,
        BlockTime:     3 * time.Second,
        ConfirmBlocks: 15,
    },
}

// 多链索引器
type MultiChainIndexer struct {
    indexers map[string]*IndexerService
}

func (m *MultiChainIndexer) Start(ctx context.Context) error {
    for chain, indexer := range m.indexers {
        go indexer.Run(ctx)
    }
    return nil
}
```

---

## 附录

### A. 术语表

| 术语 | 说明 |
|------|------|
| **Event** | 智能合约发出的日志事件 |
| **Reorg** | 区块链重组,链发生分叉 |
| **Finality** | 区块最终确定,不再回滚 |
| **ABI** | Application Binary Interface,合约接口定义 |
| **RPC** | Remote Procedure Call,区块链节点接口 |
| **Indexed Parameter** | 事件中可索引的参数 |
| **Log Index** | 交易中事件的序号 |

### B. 参考资料

- [Ethereum Yellow Paper](https://ethereum.github.io/yellowpaper/paper.pdf)
- [Go Ethereum Book](https://goethereumbook.org/)
- [GraphQL Best Practices](https://graphql.org/learn/best-practices/)
- [PostgreSQL Performance Tuning](https://wiki.postgresql.org/wiki/Performance_Optimization)
- [Microservices Patterns](https://microservices.io/patterns/)

### C. 变更历史

| 版本 | 日期 | 作者 | 变更内容 |
|------|------|------|----------|
| v1.0 | 2025-10-15 | [Your Name] | 初始版本 |

---

**文档状态:** 📝 Draft  
**审核者:** TBD  
**批准者:** TBD  
**下次审核日期:** TBD