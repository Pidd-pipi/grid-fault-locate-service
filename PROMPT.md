# 配电网故障定位与复电服务（grid-fault-locate-service）

## 一、项目概述

基于 Go 实现的配电网故障定位 Web 项目，一款后端服务，完成配网拓扑管理、故障指示器信号采集、故障区段定位、复电流程跟踪与停电统计。

项目类型：**全栈 Web 应用**（Go 后端服务 + `go:embed` 内嵌前端页面）。

## 二、业务背景与领域规则

配电网发生故障时，沿线安装的故障指示器会翻牌上报故障电流信号，变电站出线开关跳闸。调度员需要根据拓扑关系和指示器信号快速定位故障区段（两个相邻开关之间），再安排抢修并跟踪复电流程。系统维护配网拓扑（线路、开关、联络开关、指示器挂接），故障时综合多个指示器信号做定位推理，排除误报。

关键领域规则（这些规则是后续埋 bug 验证跨文件改动的核心约束，必须真实实现）：

1. 拓扑约束：线路由开关节点和区段组成，区段两端必须都是开关节点；新增/删除区段时校验拓扑连通性，成环或悬空即拒绝。
2. 故障指示器信号：指示器上报翻牌(triggered)/复位(reset)状态；同一区段多个指示器信号冲突（一个翻牌一个未翻牌）时按「上游翻牌下游未翻牌 → 故障在翻牌与未翻牌之间」定位。
3. 故障定位推理：从出线开关向下游遍历，取最后一个翻牌指示器与第一个未翻牌指示器之间的区段为候选故障区段；多候选时按信号时间排序输出。
4. 故障事件状态机：已定位(located) → 抢修中(repairing) → 已复电(restored) → 已归档(archived)；复电前必须完成故障区段隔离操作确认。
5. 复电闭环：复电操作需记录操作人、隔离区段、时间；停电时长按定位→复电计算，超 2 小时进入「长时停电」关注清单。
6. 指示器误报处理：定位时某指示器信号与其他指示器矛盾（孤立翻牌）标记为可疑并提示人工核验，不参与定位。

## 三、核心实体（≥3 个，必须贯穿全栈）

每个实体必须贯穿「数据库/存储表 → domain model → repository → service → handler → 前端 API 层 → 前端页面/组件」全链路。

| 实体 | 关键字段 | 业务动作 |
|---|---|---|
| 线路 Feeder | id、变电站、电压等级、状态 | 维护 |
| 开关节点 SwitchNode | id、线路id、类型(分段/联络/出线)、状态(合/分) | 维护、隔离 |
| 线路区段 FeederSection | id、线路id、两端开关、是否候选故障 | 定位 |
| 故障指示器 FaultIndicator | id、区段id、状态(triggered/reset)、上报时间 | 信号采集 |
| 故障事件 FaultEvent | id、线路id、候选区段、定位时间、状态、操作人 | 定位、抢修、复电 |
| 停电统计 OutageRecord | id、故障事件id、停电开始/结束、时长 | 统计 |

## 四、核心页面与 API

### 前端页面（≥4 个路由，至少 2 个页面共用同一个业务组件）

| 项目 | 说明 |
|---|---|
| / 配网总览 | 线路状态 + 故障事件 + 长时停电关注 | Feeder、FaultEvent |
| /topology 拓扑管理 | 线路拓扑图 + 区段/开关维护 | Feeder、SwitchNode、FeederSection |
| /indicators 指示器 | 指示器信号列表 + 可疑标记 | FaultIndicator |
| /faults 故障事件 | 定位结果 + 抢修/复电流转 | FaultEvent |
| /outages 停电统计 | 停电时长统计 | OutageRecord |

### 后端 REST API（与页面一一对应，命中真实业务链路）

| 项目 | 说明 |
|---|---|
| POST /api/indicators/{id}/signal | 指示器信号上报（翻牌/复位） |
| POST /api/faults/locate | 故障定位推理（综合信号 + 拓扑） |
| POST /api/faults/{id}/isolate | 隔离区段操作确认 |
| POST /api/faults/{id}/restore | 复电完成 |
| POST /api/feeders/{id}/sections | 新增区段（拓扑校验） |
| GET /api/feeders/{id}/topology | 拓扑数据 |
| GET /api/faults | 故障事件列表 |
| GET /api/outages | 停电统计 |
| GET /api/healthz | 健康检查 |

## 五、横切关注点（≥2 个）

1. 操作审计日志：定位确认、隔离、复电、拓扑变更全部留痕；触达 handler → service → audit store。
2. 长时停电扫描定时任务：每 10 分钟扫描超 2 小时未复电事件；触达 service → store → 总览。
3. 全局错误处理与统一响应格式。

## 六、共享枚举/常量（≥2 组）

枚举/常量要求前后端各自定义且保持一致，README 中列出所有出现位置。

1. 开关状态 SwitchStatus：closed / open；指示器状态 IndicatorStatus：triggered / reset。
2. 故障事件状态 FaultStatus：located / repairing / restored / archived。
3. 开关类型 SwitchType：sectionalizer / tie / feeder_outlet。

## 七、共享前端组件与 hooks（组件 ≥3 个、hooks ≥2 个）

### 共享组件（放 `web/components/`）

1. TopologyGraph：拓扑图组件，被拓扑管理与故障页共用。
2. FaultCard：故障事件卡片，被总览与故障页共用。
3. IndicatorTable：指示器信号表格，被指示器页与定位弹窗共用。

### 自定义 hooks（放 `web/hooks/`）

1. useFeeders()：线路列表，被总览与拓扑页共用。
2. useFaults(filter)：故障事件，被故障页与总览共用。

## 八、后端中间件（≥2 个）

1. auditLogger：审计日志中间件。
2. errorHandler：统一错误/panic 处理中间件。
3. requestID：trace id 注入中间件。

## 九、技术要求

- 语言：**Go 1.23**（go.mod 声明 `go 1.23`，module 路径 `example.com/grid-fault-locate-service`）
- 运行：`go run .` 默认监听 `8080`，支持 `PORT` 环境变量覆盖
- 存储：SQLite（`modernc.org/sqlite` 纯 Go 驱动，CGO 关闭）或内置内存仓储 + JSON 文件持久化，二选一，必须可重复构建、无外部服务依赖
- 前端：纯原生 HTML/CSS/JS，`go:embed` 内嵌 `web/` 静态资源，禁止引入外部 CDN 依赖（离线可跑）
- 服务入口：`GET /healthz` 返回 200；页面 `GET /` 可访问
- 根目录必须包含 `runtime_smoke.json`：`mode: service` + `start: go run .` + `ready_url: /healthz`；`project_intro` 一句话简介必须包含项目类型（如「基于 Go 实现的XXX Web 项目，一款后端服务，完成……」）
- 根目录必须包含 `README.md`：项目说明、目录结构、运行与测试命令、环境变量说明
- 构建：`go build ./...` 与 `go test ./...` 必须全部通过（基线干净、无 bug）

## 十、文件结构强制清单（规模目标：≥2000 行 Go 功能代码、≥20 个 `.go` 文件）

```
backend/
├── go.mod
├── main.go
├── config/
│   └── config.go            # 电压等级、长时停电阈值
├── domain/
│   ├── feeder.go            # 线路实体
│   ├── switch.go            # 开关节点
│   ├── section.go           # 线路区段 + 拓扑校验
│   ├── indicator.go         # 故障指示器信号
│   ├── fault.go             # 故障事件状态机
│   └── outage.go            # 停电统计
├── store/
│   ├── feeder_store.go
│   ├── switch_store.go
│   ├── section_store.go
│   ├── indicator_store.go
│   ├── fault_store.go
│   ├── outage_store.go
│   └── audit_store.go
├── service/
│   ├── topology_service.go  # 拓扑维护 + 连通性校验
│   ├── signal_service.go    # 指示器信号
│   ├── locate_service.go    # 故障定位推理
│   ├── fault_service.go     # 隔离/复电
│   ├── outage_service.go    # 停电统计
│   ├── sweeper.go           # 长时停电扫描
│   └── audit_service.go
├── httpapi/
│   ├── router.go
│   ├── feeder_handler.go
│   ├── topology_handler.go
│   ├── indicator_handler.go
│   ├── fault_handler.go
│   ├── outage_handler.go
│   └── health_handler.go
├── middleware/
│   ├── audit.go
│   ├── error_handler.go
│   └── request_id.go
└── web/
    ├── index.html
    ├── app.js
    ├── style.css
    ├── components/
    └── hooks/
```

**严禁合并职责到单一文件**：handler、service、repository、domain 必须分层；禁止把所有逻辑塞进 `main.go` 或一个 `handlers.go`。目标规模下限 2000 行 / 20 个 `.go` 文件，实际建议做到 3000 行以上 / 30 个文件以上，保证每个业务模块（实体、状态机、联动、报表）都有独立文件。

## 十一、运行、测试与交付要求

1. `go build ./...` 通过；`go test ./...` 全绿（含各业务模块的单元测试，测试文件不计入规模）。
2. `go run .` 后 `GET /healthz` 返回 200，前端页面 `GET /` 可打开且核心接口可用。
3. 每个核心业务动作都要有可复现的输入（API 请求/页面操作），方便后续构造缺陷与验证命令。
4. 代码中不得出现任何「故意埋错」「TODO bug」类注释；交付为干净基线。

## 十二、质量红线

1. **天然多文件、多层耦合**：任何一个小改动（如给某状态新增一个合法迁移）都应触达 3-5 个文件（domain + repository + service + handler + 前端组件 + 枚举定义）。
2. 业务规则必须具体、可验证：状态机迁移表、联动逻辑、校验边界、生命周期管理必须真实存在，禁止空壳 CRUD。
3. 本项目用于评测跨文件协同改动能力，禁止做成本目录、对账/财务、库存盘点、电商订单、预约挂号、工单客服、数据可视化报表类业务。
4. 前端页面必须真实消费后端接口，禁止纯静态假页面。

---
*生成说明：本提示词面向 Go 标注数据流水线 2000 行档位，主题已对照禁选题材清单核验。*
