# Smart Contract Event Indexer - 项目需求文档
## Product Requirements Document (PRD)

---

## 1. 项目概述 (Project Overview)

### 1.1 项目简介
构建一个高性能的智能合约事件索引服务，实时监听区块链上的智能合约事件，解析并存储到数据库中，通过GraphQL/REST API提供快速查询服务。

**项目定位**：🎯 **Portfolio/技能展示项目**
- 展示 Web3 开发技能
- 展示微服务架构能力
- 展示系统设计和工程实践
- **优先使用免费服务降低成本**

**核心价值**：
- 解决直接查询区块链慢、成本高的问题
- 为DApp提供快速的历史数据查询能力
- 支持复杂的数据聚合和分析查询

### 1.2 目标用户
- DApp前端开发者（需要查询历史交易数据）
- DeFi分析师（需要分析协议数据）
- Web3数据产品（需要可靠的数据源）

### 1.3 成功指标

**性能指标（可配置确认策略）**：

| 模式 | 确认块数 | 索引延迟 | 数据准确率 | 适用场景 |
|------|---------|---------|-----------|---------|
| **实时模式** | 1 块 | < 15秒 | ~99% | Demo、非关键应用 |
| **平衡模式** (推荐) | 6 块 | < 90秒 | ~99.99% | 大多数生产应用 |
| **安全模式** | 12 块 | < 150秒 | ~99.9999% | 金融、审计系统 |

**其他指标**：
- API响应时间 < 200ms (P95)
- 支持处理 1000+ events/second
- 系统可用性 99.9%

**关于确认策略的说明**：
- **默认使用平衡模式（6块确认）**：在速度和安全间取得最佳平衡
- Ethereum Mainnet: 6块 ≈ 72秒（在可接受范围内）
- 允许每个合约配置不同的确认策略

**Portfolio部署建议**：
- **RPC节点**: 使用免费服务（Alchemy 300M CU/月 或 Infura 100K请求/天）
- **成本控制**: 实现请求缓存和批量获取减少RPC调用
- **扩展性**: 架构设计支持未来升级到付费服务

---

## 2. 核心功能需求 (Core Features)

### 2.1 区块链监听模块 (Blockchain Listener)
**功能描述**：实时监听指定智能合约的事件

**详细需求**：
- ✅ 支持监听多个智能合约地址
- ✅ 支持配置监听的事件类型（通过ABI）
- ✅ 自动处理chain reorganization（区块重组）
- ✅ 断点续传机制（服务重启后从上次位置继续）
- ✅ 支持批量获取历史事件（backfill）
- ✅ 连接失败自动重连机制

**技术实现点**：
- 使用WebSocket订阅新区块
- 每个区块确认后获取事件日志
- 维护最近N个区块的缓存处理reorg

### 2.2 事件解析与存储 (Event Parsing & Storage)
**功能描述**：解析事件数据并规范化存储

**详细需求**：
- ✅ 自动解析event参数（根据ABI）
- ✅ 提取indexed和non-indexed参数
- ✅ 类型转换（BigNumber → string, bytes → hex）
- ✅ 存储完整的交易上下文（txHash, blockNumber, timestamp）
- ✅ 支持事件数据关联（同一笔交易的多个事件）
- ✅ 数据去重（防止重复索引）

**数据存储要求**：
- 原始事件数据保持完整性
- 建立适当索引加速查询
- 支持按时间、区块号、地址查询

### 2.3 GraphQL API (Primary API)
**功能描述**：提供灵活的GraphQL查询接口

**核心Query**：
```graphql
# 查询特定合约的事件
query GetEvents {
  events(
    contractAddress: "0x..."
    eventName: "Transfer"
    fromBlock: 12345
    toBlock: 12500
    first: 20
  ) {
    edges {
      node {
        id
        eventName
        blockNumber
        timestamp
        transactionHash
        args {
          key
          value
        }
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}

# 查询特定地址的相关事件
# 注意：MVP阶段使用GIN索引，性能优化阶段迁移到专用地址表
query GetUserActivity {
  eventsByAddress(
    address: "0xUserAddress"
    first: 50
  ) {
    # ... 类似结构
  }
}

# 聚合查询示例
query GetStats {
  transferStats(contractAddress: "0x...") {
    totalVolume
    uniqueSenders
    totalTransactions
  }
}
```

**Mutation**（管理功能）：
```graphql
# 添加监听合约（幂等操作）
mutation AddContract {
  addContract(
    address: "0x..."
    abi: "..."
    startBlock: 12345
    confirmBlocks: 6  # 可选，默认6块（平衡模式）
  ) {
    success
    contractId
    isNew  # 标识是新建还是已存在
    message
  }
}
```

**实现注意事项**：
- ✅ 所有 Mutation 必须实现幂等性
- ✅ `eventsByAddress` 在MVP阶段使用 GIN 索引，Phase 3 优化为专用表
- ✅ 大数字类型（uint256）统一返回 String 防止精度丢失

### 2.4 REST API (Alternative)
**功能描述**：提供简单的REST查询接口

**核心Endpoints**：
```
GET  /api/events
     ?contract=0x...&eventName=Transfer&limit=50

GET  /api/events/:txHash
     返回特定交易的所有事件

GET  /api/contracts/:address/stats
     返回合约统计信息

POST /api/contracts
     添加新的监听合约
```

### 2.5 管理后台功能
**功能描述**：管理索引配置和监控状态

**核心功能**：
- 添加/删除监听合约
- 查看索引状态（当前区块、延迟）
- 触发历史数据回填
- 查看错误日志
- 服务健康检查

---

## 3. 技术架构 (Technical Architecture)

### 3.1 系统架构图
```
┌─────────────────────────────────────────────────┐
│                   Client Layer                   │
│  (DApp Frontend / Analytics Dashboard / Mobile) │
└─────────────────┬───────────────────────────────┘
                  │ GraphQL/REST
┌─────────────────▼───────────────────────────────┐
│              API Gateway (Go)                    │
│  - GraphQL Server (gqlgen)                      │
│  - REST API (gin/fiber)                         │
│  - Authentication & Rate Limiting               │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│           Service Layer (Go)                     │
│                                                  │
│  ┌────────────────┐  ┌──────────────────────┐  │
│  │ Event Indexer  │  │  Query Service       │  │
│  │ - Listener     │  │  - Data Aggregation  │  │
│  │ - Parser       │  │  - Cache Layer       │  │
│  │ - Validator    │  │                      │  │
│  └────────┬───────┘  └───────────┬──────────┘  │
└───────────┼──────────────────────┼──────────────┘
            │                      │
  ┌─────────▼──────────┐  ┌───────▼──────────┐
  │  Blockchain Node   │  │   PostgreSQL     │
  │  (Geth/Infura)    │  │   - Events       │
  │                   │  │   - Contracts    │
  └───────────────────┘  │   - Metadata     │
                         └──────────────────┘
  ┌───────────────────┐  ┌──────────────────┐
  │  Redis Cache      │  │   Monitoring     │
  │  - Query Cache    │  │   - Prometheus   │
  │  - Block State    │  │   - Grafana      │
  └───────────────────┘  └──────────────────┘
```

### 3.2 核心技术栈

**Backend**:
- **Go 1.21+**: 主要开发语言
- **go-ethereum (geth)**: Web3客户端库
- **gqlgen**: GraphQL服务器生成器
- **Gin/Fiber**: HTTP框架（如需REST）

**Database**:
- **PostgreSQL 15**: 主数据库
  - 存储事件数据
  - 使用JSONB存储event args
  - 使用btree_gin索引优化查询
- **Redis 7**: 缓存层
  - 查询结果缓存
  - 区块状态缓存

**Infrastructure**:
- **Docker**: 容器化
- **Docker Compose**: 本地开发环境
- **AWS ECS/EKS**: 生产部署（可选）
- **Prometheus + Grafana**: 监控

**Testing**:
- Go testing framework
- Testcontainers: 集成测试
- Mock Ethereum node

---

## 4. 数据模型 (Data Models)

### 4.1 数据库Schema

```sql
-- 监听的合约配置
CREATE TABLE contracts (
    id SERIAL PRIMARY KEY,
    address VARCHAR(42) NOT NULL UNIQUE,
    name VARCHAR(100),
    abi JSONB NOT NULL,
    start_block BIGINT NOT NULL,
    current_block BIGINT NOT NULL DEFAULT 0,
    confirm_blocks INTEGER NOT NULL DEFAULT 6, -- 确认块数：1(实时), 6(平衡), 12(安全)
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- 约束：确认块数必须在合理范围内
    CONSTRAINT valid_confirm_blocks CHECK (confirm_blocks >= 1 AND confirm_blocks <= 64)
);

-- 事件数据主表
CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    contract_id INTEGER REFERENCES contracts(id),
    contract_address VARCHAR(42) NOT NULL,
    event_name VARCHAR(100) NOT NULL,
    block_number BIGINT NOT NULL,
    block_timestamp TIMESTAMP NOT NULL,
    transaction_hash VARCHAR(66) NOT NULL,
    transaction_index INTEGER NOT NULL,
    log_index INTEGER NOT NULL,
    args JSONB NOT NULL, -- 事件参数
    raw_log JSONB, -- 原始log数据
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- 确保唯一性
    UNIQUE(transaction_hash, log_index)
);

-- 索引优化
CREATE INDEX idx_events_contract ON events(contract_address, event_name);
CREATE INDEX idx_events_block ON events(block_number DESC);
CREATE INDEX idx_events_timestamp ON events(block_timestamp DESC);
CREATE INDEX idx_events_tx ON events(transaction_hash);
CREATE INDEX idx_events_args ON events USING GIN(args);

-- 地址参数优化表 (针对频繁的 eventsByAddress 查询)
-- MVP阶段可选，性能优化阶段实现
CREATE TABLE event_addresses (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT REFERENCES events(id) ON DELETE CASCADE,
    param_name VARCHAR(50) NOT NULL, -- 'from', 'to', 'owner' etc.
    address VARCHAR(42) NOT NULL,
    
    -- 高效查询索引
    UNIQUE(event_id, param_name)
);

CREATE INDEX idx_event_addresses_lookup ON event_addresses(address, param_name);
CREATE INDEX idx_event_addresses_event ON event_addresses(event_id);

-- 索引状态追踪
CREATE TABLE indexer_state (
    contract_id INTEGER PRIMARY KEY REFERENCES contracts(id),
    last_indexed_block BIGINT NOT NULL,
    last_indexed_at TIMESTAMP DEFAULT NOW(),
    is_syncing BOOLEAN DEFAULT false,
    error_message TEXT
);
```

### 4.2 Go数据结构

```go
// Event represents a blockchain event
type Event struct {
    ID              int64
    ContractID      int32
    ContractAddress string
    EventName       string
    BlockNumber     int64
    BlockTimestamp  time.Time
    TxHash          string
    TxIndex         int32
    LogIndex        int32
    Args            map[string]interface{}
    RawLog          json.RawMessage
    CreatedAt       time.Time
}

// Contract represents a monitored smart contract
type Contract struct {
    ID            int32
    Address       string
    Name          string
    ABI           json.RawMessage
    StartBlock    int64
    CurrentBlock  int64
    ConfirmBlocks int32           // 确认块数：1(实时), 6(平衡), 12(安全)
    IsActive      bool
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

---

## 5. API设计详细说明

### 5.1 GraphQL Schema

```graphql
type Event {
  id: ID!
  contractAddress: String!
  eventName: String!
  blockNumber: Int!
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

type EventConnection {
  edges: [EventEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type Query {
  # 基础查询
  events(
    contractAddress: String
    eventName: String
    fromBlock: Int
    toBlock: Int
    first: Int
    after: String
  ): EventConnection!
  
  # 按交易查询
  eventsByTransaction(txHash: String!): [Event!]!
  
  # 按地址查询（如果event中包含address字段）
  eventsByAddress(
    address: String!
    first: Int
    after: String
  ): EventConnection!
  
  # 合约统计
  contractStats(address: String!): ContractStats!
}

type ContractStats {
  totalEvents: Int!
  latestBlock: Int!
  indexerDelay: Int! # 秒
}
```

---

## 6. 开发计划 (Development Roadmap)

### Phase 1: MVP (Week 1-2)
- [ ] 项目初始化（Go modules, 目录结构）
- [ ] 连接Ethereum节点（go-ethereum）+ RPC fallback机制 🔴
- [ ] 实现基础事件监听器
- [ ] **实现可配置确认块策略**（默认6块，支持1/6/12块）
- [ ] PostgreSQL数据模型设计与实现（GIN索引方案 + confirm_blocks字段）
- [ ] 简单的REST API（查询events）
- [ ] Docker Compose开发环境
- [ ] **关键**：实现Mutation幂等性 + Chain reorg基础处理

**交付物**：可以监听一个合约并通过API查询事件
**重点**：RPC稳定性 > 功能完整性
**性能目标**：平衡模式（6块，~72秒延迟）

### Phase 2: 核心功能完善 (Week 3)
- [ ] 实现GraphQL API（gqlgen）
- [ ] 添加pagination支持
- [ ] 完善chain reorg处理逻辑（支持所有确认策略：1/6/12块）🔴
- [ ] 添加Redis缓存层
- [ ] 配置管理系统（添加/删除合约，支持设置确认块数，带幂等性）
- [ ] 基础监控（Prometheus metrics）

**交付物**：生产级的索引服务（核心功能）
**性能目标**：默认使用平衡模式（6块，~90秒），P95 < 300ms

### Phase 3: 性能优化 (Week 4)
- [ ] **评估并实现** `event_addresses` 表（如果查询P95 > 500ms）
- [ ] 历史数据回填功能
- [ ] 批处理优化（动态batch size）
- [ ] 完整监控和告警（Grafana dashboards，包括确认策略监控）
- [ ] 管理后台UI（可选）
- [ ] 单元测试与集成测试（覆盖率 > 75%，测试所有确认策略）
- [ ] 压力测试（k6）

**交付物**：生产就绪系统
**性能目标**：
- 平衡模式（6块）: ~72秒延迟，P95 < 200ms
- 实时模式（1块）: ~12秒延迟（用于Demo）
- 安全模式（12块）: ~144秒延迟（用于金融应用）

### Phase 4: 部署与文档 (Week 5)
- [ ] **免费云部署配置** (Railway.app $5/月 或 Oracle Cloud 永久免费)
- [ ] CI/CD pipeline设置 (GitHub Actions 免费)
- [ ] API文档（GraphQL Playground + README）
- [ ] 架构决策文档（ADR）- 记录技术权衡
- [ ] 性能测试报告
- [ ] Demo视频录制

**交付物**：可展示的项目portfolio

**Portfolio 部署成本目标**: **$0-5/月**
- Railway.app: $5/月 (或使用 $5 免费额度)
- Supabase PostgreSQL: 免费 (500MB)
- Upstash Redis: 免费 (10K 命令/天)
- Alchemy RPC: 免费 (300M CU/月)
- BetterUptime 监控: 免费

**关键里程碑检查点**:
- Week 2 结束：能稳定索引至少1个合约
- Week 3 结束：GraphQL API可用，通过基础功能测试
- Week 4 结束：性能达标，可演示
- Week 5 结束：文档完善，ready for showcase

---

## 7. 性能要求与优化策略

### 7.1 性能目标

**索引延迟（取决于确认策略）**：
- **实时模式** (1块): < 15秒
- **平衡模式** (6块): < 90秒 ← 默认
- **安全模式** (12块): < 150秒

**API性能**：
- API响应时间: P50 < 50ms, P95 < 200ms
- 吞吐量: 支持 1000+ events/second
- 可用性: 99.9% uptime

### 7.2 优化策略
1. **批量处理**: 批量插入事件到数据库
2. **连接池**: 数据库连接池优化
3. **缓存策略**: 
   - 热点查询使用Redis缓存（TTL 30s）
   - 区块确认状态缓存
4. **并发处理**: 使用goroutine并发处理多个合约
5. **数据库索引**: 为常用查询字段建立索引

---

## 8. 测试策略

### 8.1 测试范围
- **单元测试**: 核心逻辑（事件解析、数据转换）
- **集成测试**: 
  - 使用Ganache本地链测试
  - 使用Testcontainers测试数据库交互
- **E2E测试**: 完整的监听->存储->查询流程
- **性能测试**: 使用k6测试API性能

### 8.2 测试覆盖率目标
- 核心业务逻辑: > 80%
- API handlers: > 70%

---

## 9. 风险与挑战

| 风险 | 影响 | 优先级 | 缓解方案 |
|------|------|--------|----------|
| RPC节点限流/不稳定 | 索引延迟增加，可能掉数据 | 🔴 P0 | **Portfolio**: 免费RPC (Alchemy/Infura) + 多节点fallback + 智能重试 + 请求缓存。**生产环境**: 考虑付费RPC ($49+/月) |
| Chain reorg处理不当 | 数据不一致，用户看到错误数据 | 🔴 P0 | **可配置确认策略**：默认6块（平衡），可选1块（快速）或12块（安全）+ 实现reorg检测和回滚逻辑 |
| `eventsByAddress` 性能瓶颈 | 查询变慢，用户体验差 | 🟡 P1 | MVP使用GIN索引，Phase 3引入专用地址表 |
| 数据库性能瓶颈 | 写入变慢，索引延迟增加 | 🟡 P1 | 批量插入 + 连接池优化 + 分表策略（后期） |
| 内存占用过高 | 服务OOM崩溃 | 🟢 P2 | 限制批处理大小 + 事件缓冲区上限 + 定期GC |
| Mutation非幂等性 | API误用导致重复数据 | 🟢 P2 | 实现幂等性设计 + 数据库UNIQUE约束 |

**风险应对优先级说明**:
- 🔴 P0: 必须在MVP阶段解决，否则系统不可用
- 🟡 P1: 应在Phase 2-3解决，影响用户体验
- 🟢 P2: 可在生产优化阶段解决

**关键决策 (Portfolio项目)**:
- **RPC服务**: 免费层完全足够 Portfolio 展示
  - Alchemy Free: 300M 计算单元/月 (约等于 5-10 个合约的索引需求)
  - Infura Free: 100K 请求/天 (足够小规模测试)
  - 实现多节点 fallback 确保可靠性
- **优化策略**: 批量请求 + 缓存减少 RPC 调用 99%
- **扩展路径**: 架构支持无缝升级到付费 RPC ($49-99/月) 用于生产部署

---

## 10. 技术权衡与实现策略 (Technical Trade-offs)

### 10.1 地址查询优化策略

**问题**: `eventsByAddress` 查询性能

**方案对比**:

| 方案 | 优点 | 缺点 | 建议阶段 |
|------|------|------|----------|
| **GIN索引 (JSONB)** | 实现简单，灵活 | 查询较慢，全表扫描风险 | MVP阶段 |
| **专用地址表** | 查询极快，精确索引 | 存储冗余，写入开销+50% | 优化阶段 |
| **物化视图** | 平衡性能与存储 | 需要定期刷新，复杂度增加 | 大规模场景 |

**推荐实施路径**:
1. **Phase 1-2**: 使用 GIN 索引，验证功能
2. **Phase 3**: 监控查询性能，如果 P95 > 500ms，引入 `event_addresses` 表
3. **生产优化**: 根据实际查询模式决定是否需要物化视图

**实现细节**:
```go
// Phase 1: GIN索引查询 (MVP)
SELECT * FROM events 
WHERE args @> '{"from": "0x..."}'::jsonb;

// Phase 3: 专用表查询 (优化)
SELECT e.* FROM events e
JOIN event_addresses ea ON e.id = ea.event_id
WHERE ea.address = '0x...' 
  AND ea.param_name IN ('from', 'to');
```

### 10.2 事件参数类型处理

**问题**: GraphQL中如何表示不同类型的事件参数

**当前方案**: 统一使用 `String` 类型
```graphql
type EventArg {
  key: String!
  value: String!  # 所有类型都是String
  type: String!   # "uint256", "address", "bool"等
}
```

**权衡分析**:

✅ **优点**:
- 类型安全：uint256 不会丢失精度
- 简单：前端统一处理字符串
- 灵活：支持任意复杂类型

⚠️ **缺点**:
- 前端需要类型转换（但这是合理的）
- 无法利用GraphQL的类型优势

**替代方案**: 使用 GraphQL Union 类型
```graphql
type EventArg {
  key: String!
  value: EventValue!
}

union EventValue = StringValue | IntValue | BoolValue | AddressValue

type StringValue { value: String! }
type IntValue { value: String! }  # 仍用String防止精度丢失
type BoolValue { value: Boolean! }
type AddressValue { value: String!, checksummed: String! }
```

**建议**: MVP阶段使用简单的 String 方案，除非有明确需求

### 10.3 RPC节点策略

**问题**: <5秒延迟严重依赖RPC稳定性

**缓解策略（优先级排序）**:

1. **使用专用RPC节点** ⭐⭐⭐⭐⭐
   - Alchemy Growth ($49/月) 或 Infura Growth ($50/月)
   - 避免公共端点的限流问题
   - 成本收益比最高

2. **实现智能重试和Fallback** ⭐⭐⭐⭐⭐
   ```go
   type RPCProvider struct {
       Primary   *ethclient.Client
       Fallback  []*ethclient.Client
       Current   int
   }
   
   func (p *RPCProvider) CallWithFallback(ctx context.Context, fn func(*ethclient.Client) error) error {
       // 尝试主节点
       if err := fn(p.Primary); err == nil {
           return nil
       }
       
       // 依次尝试fallback节点
       for _, fb := range p.Fallback {
           if err := fn(fb); err == nil {
               return nil
           }
       }
       return errors.New("all RPC endpoints failed")
   }
   ```

3. **请求批处理** ⭐⭐⭐⭐
   - 使用 `eth_getLogs` 批量获取事件
   - 单次请求覆盖多个区块（如10-50个）
   - 减少API调用次数

4. **本地Geth节点** ⭐⭐⭐
   - 零延迟，无限流
   - 但需要维护成本（~1TB存储，Archive Node更大）
   - 适合高流量生产环境

**MVP阶段建议**: 方案1 + 方案2，成本低且可靠

### 10.4 Mutation幂等性设计

**问题**: `AddContract` 重复调用应该如何处理

**推荐实现**:
```go
func (r *mutationResolver) AddContract(ctx context.Context, input AddContractInput) (*AddContractPayload, error) {
    // 检查是否已存在
    existing, err := r.DB.GetContractByAddress(input.Address)
    if err == nil {
        // 已存在，返回现有记录
        return &AddContractPayload{
            Success:    true,
            ContractID: existing.ID,
            IsNew:      false,
            Message:    "Contract already exists",
        }, nil
    }
    
    // 不存在，创建新记录
    contract, err := r.DB.CreateContract(input)
    if err != nil {
        return &AddContractPayload{
            Success: false,
            Message: err.Error(),
        }, nil
    }
    
    return &AddContractPayload{
        Success:    true,
        ContractID: contract.ID,
        IsNew:      true,
        Message:    "Contract added successfully",
    }, nil
}
```

**关键点**:
- ✅ 不抛出错误，优雅处理
- ✅ 返回 `isNew` 标识
- ✅ 数据库层面使用 `UNIQUE` 约束防止重复

### 10.5 高吞吐链适配

**问题**: Solana/BSC等高吞吐链的特殊考虑

**Ethereum vs 高吞吐链对比**:

| 特性 | Ethereum | Solana | BSC |
|------|----------|--------|-----|
| 出块时间 | 12秒 | 0.4秒 | 3秒 |
| TPS | ~15 | ~3000 | ~160 |
| Finality | 12-32块 | 32块 | 15块 |

**架构调整建议**:
1. **批处理窗口**: 从1个区块增加到10-50个区块
2. **缓冲区**: 增大事件缓冲队列
3. **并发度**: 适当增加goroutine数量
4. **延迟目标**: 调整为 <10秒（更现实）

**代码层面预留扩展点**:
```go
type ChainConfig struct {
    ChainID        int64
    BlockTime      time.Duration
    BatchSize      int
    ConfirmBlocks  int
}

var chainConfigs = map[string]ChainConfig{
    "ethereum": {ChainID: 1, BlockTime: 12*time.Second, BatchSize: 10, ConfirmBlocks: 12},
    "bsc":      {ChainID: 56, BlockTime: 3*time.Second, BatchSize: 50, ConfirmBlocks: 15},
    "polygon":  {ChainID: 137, BlockTime: 2*time.Second, BatchSize: 100, ConfirmBlocks: 128},
}
```

---

## 11. 项目展示要点 (Portfolio Highlights)

展示这个项目时，重点突出：

✅ **系统设计能力**: 
   - 事件驱动架构
   - 高并发处理（goroutines）
   - 缓存策略

✅ **区块链技术栈**:
   - 熟悉Web3开发
   - 理解区块链特性（reorg, finality）
   - 智能合约交互

✅ **工程最佳实践**:
   - 清晰的代码结构
   - 完善的错误处理
   - 监控和可观测性
   - Docker化部署

✅ **数据工程**:
   - 高效的数据管道
   - 数据建模与索引优化
   - GraphQL API设计

---

## 附录：推荐的项目结构

```
event-indexer/
├── cmd/
│   ├── indexer/        # 索引服务入口
│   └── api/            # API服务入口
├── internal/
│   ├── blockchain/     # 区块链交互
│   ├── parser/         # 事件解析
│   ├── storage/        # 数据存储
│   ├── api/            # API handlers
│   └── config/         # 配置管理
├── pkg/
│   └── models/         # 共享数据模型
├── migrations/         # 数据库迁移
├── graphql/
│   └── schema.graphql  # GraphQL schema
├── docker-compose.yml
├── Dockerfile
└── README.md
```

---

**文档版本**: v1.0  
**最后更新**: 2025-10-12