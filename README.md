# 配电网故障定位与复电服务（grid-fault-locate-service）

基于 Go 实现的全栈 Web 应用：Go 后端服务 + `go:embed` 内嵌原生前端（HTML/CSS/JS，零外部 CDN、离线可跑）。完成配网拓扑管理、故障指示器信号采集、故障区段定位、复电流程跟踪与停电统计。

## 一、核心业务规则

1. **拓扑约束**：线路由开关节点和区段组成，区段两端必须是开关节点；新增/删除区段时校验拓扑连通性，成环或悬空即拒绝（`service/topology_service.go` + `domain/section.go` 纯函数）。
2. **指示器信号冲突**：同一区段多个指示器信号冲突（一个翻牌一个未翻牌）时，按「上游翻牌下游未翻牌 → 故障在翻牌与未翻牌之间」定位。
3. **故障定位推理**：从出线开关向下游遍历，取最后一个翻牌指示器与第一个未翻牌指示器之间的区段为候选故障区段；多候选时按信号时间排序输出（`service/locate_service.go`）。
4. **故障事件状态机**：`located → repairing → restored → archived`；复电前必须完成故障区段隔离操作确认（`domain/fault.go` 迁移表）。
5. **复电闭环**：复电记录操作人、隔离区段、时间；停电时长按 定位→复电 计算，超 2 小时进入「长时停电」关注清单（`service/outage_service.go`、`service/sweeper.go`）。
6. **指示器误报处理**：孤立翻牌（所在区段与相邻区段均无其他翻牌且不在候选区）标记为可疑并提示人工核验，不参与定位。

## 二、架构与分层

| 层 | 目录 | 职责 |
|---|---|---|
| 入口 | `main.go` | 加载并校验配置、初始化 `log/slog`、组装依赖、启动定时任务与 HTTP 服务、优雅关闭 |
| 配置 | `config/` | 环境变量覆盖 + `Config.Validate()`，维护电压等级等共享常量 |
| 领域 | `domain/` | 实体模型、共享枚举、状态机、拓扑纯函数；不依赖存储与 HTTP |
| 存储 | `store/` | 内存仓储 + JSON 原子持久化（临时文件 → fsync → rename）；读写锁串行化，损坏文件自动备份降级 |
| 业务 | `service/` | 拓扑维护、信号采集、故障定位、复电流转、停电统计、审计、长时停电扫描 |
| HTTP | `httpapi/` | REST 路由与 handler，统一响应格式，分页、输入校验、健康检查 |
| 中间件 | `middleware/` | requestID、结构化访问日志、recover、安全响应头、HTTP 审计 |
| 前端 | `web/` | 原生 HTML/CSS/JS，hash 路由；组件与 hooks 共享复用 |

## 三、目录结构

```
grid-fault-locate-service/
├── go.mod / main.go            # 模块与入口
├── runtime_smoke.json          # 冒烟配置（保留契约）
├── Dockerfile / Makefile / .dockerignore
├── config/config.go            # 配置、环境变量覆盖、Validate
├── domain/                     # 领域模型与纯业务规则
│   ├── feeder.go switch.go section.go indicator.go fault.go outage.go audit.go errors.go
├── store/                      # 内存仓储 + JSON 原子持久化
│   ├── store.go json.go feeder_store.go switch_store.go section_store.go
│   ├── indicator_store.go fault_store.go outage_store.go audit_store.go
├── service/                    # 业务服务层
│   ├── topology_service.go     # 拓扑维护 + 连通性校验
│   ├── signal_service.go       # 指示器信号采集/可疑标记
│   ├── locate_service.go       # 故障定位推理
│   ├── fault_service.go        # 抢修/隔离/复电/归档（含开关联动）
│   ├── outage_service.go       # 停电统计
│   ├── sweeper.go              # 长时停电扫描
│   ├── audit_service.go overview_service.go bootstrap.go
├── httpapi/                    # REST API 处理器、路由、响应封装、分页
├── middleware/                 # requestID / RequestLog / Recover / SecurityHeaders / Audit
└── web/                        # go:embed 内嵌前端
    ├── index.html app.js style.css enums.js
    ├── components/             # TopologyGraph.js FaultCard.js IndicatorTable.js
    └── hooks/                  # useFeeders.js useFaults.js useTopology.js api.js
```

## 四、本地运行

```bash
# 依赖：Go 1.23+，无需第三方模块
go run .

# 指定端口与数据文件
PORT=19009 DATA_FILE=./data/demo.json go run .
```

首次启动自动写入演示数据（`service/bootstrap.go`：两条线路 + 开关/区段/指示器），便于页面展示与故障演练。持久化关闭方式：`PERSIST=false go run .`。

## 五、测试与质量检查

```bash
make fmt        # gofmt
make vet        # go vet ./...
make test       # go test ./...
make race       # go test -race ./...
make build      # CGO_ENABLED=0 go build -o bin/grid-fault-locate-service .
```

验证要求：

```bash
go build ./...
go vet ./...
gofmt -l .
go test ./...
go test -race ./...
```

## 六、Docker 部署

```bash
# 构建镜像（多阶段：golang:1.23-alpine -> alpine:3.20）
docker build -t grid-fault-locate-service:latest .

# 运行容器（默认 8080，数据持久化到宿主机 ./data）
docker run --rm -p 8080:8080 \
  -v "$PWD/data:/data" \
  -e PORT=8080 \
  grid-fault-locate-service:latest

# 或使用 Makefile
make docker-build
make docker-run
```

镜像特性：

- `CGO_ENABLED=0` 静态二进制，`alpine:3.20` 运行；
- 非 root 用户 `app` 运行；
- `EXPOSE 8080`，尊重容器内 `PORT` 环境变量；
- 内置 `HEALTHCHECK` 探测 `/healthz`。

## 七、环境变量

| 变量 | 默认 | 说明 |
|---|---|---|
| `PORT` | `8080` | HTTP 监听端口，范围 1–65535 |
| `DATA_FILE` | `data/grid-fault-locate-data.json` | JSON 持久化文件路径 |
| `PERSIST` | `true` | `false`/`0`/`no`/`off` 关闭持久化，并清空 `DATA_FILE` |
| `LOG_LEVEL` | `info` | 结构化日志级别：`debug`/`info`/`warn`/`error` |
| `LONG_OUTAGE_MINUTES` | `120` | 长时停电判定阈值（分钟） |
| `SWEEP_INTERVAL` | `10m` | 长时停电扫描周期（Go duration 格式） |
| `REQUEST_BODY_LIMIT` | `1048576` | 请求体大小上限（字节，1 MiB） |

配置在启动时由 `config.Load()` 读取环境变量并调用 `Validate()` 校验，非法配置会拒绝启动。

## 八、REST API

统一响应：`{"code":0,"message":"ok","data":...}`；业务错误 code 非 0，由 `middleware/error_handler.go` 映射 HTTP 状态码。分页 list 接口额外返回顶层 `total`：

```json
{"code":0,"message":"ok","data":[...],"total":3}
```

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/healthz` `/api/healthz` | 健康检查 |
| GET | `/readyz` `/api/readyz` | 就绪检查 |
| GET | `/api/feeders` | 线路列表（`?status=&limit=&offset=`） |
| POST | `/api/feeders` | 新增线路 |
| GET | `/api/feeders/{id}` | 线路详情 |
| PUT | `/api/feeders/{id}` | 更新线路 |
| DELETE | `/api/feeders/{id}` | 删除线路 |
| GET | `/api/feeders/{id}/topology` | 拓扑数据 |
| POST | `/api/feeders/{id}/switches` | 新增开关 |
| PUT | `/api/feeders/{id}/switches/{switchId}` | 更新开关 |
| POST | `/api/feeders/{id}/switches/{switchId}/toggle` | 分/合闸 |
| DELETE | `/api/feeders/{id}/switches/{switchId}` | 删除开关 |
| POST | `/api/feeders/{id}/sections` | 新增区段（拓扑校验） |
| PUT | `/api/feeders/{id}/sections/{sectionId}` | 更新区段 |
| DELETE | `/api/feeders/{id}/sections/{sectionId}` | 删除区段（连通性校验） |
| GET | `/api/indicators` | 指示器列表（`?feederId=&sectionId=&suspicious=&triggered=&limit=&offset=`） |
| POST | `/api/indicators` | 新增指示器 |
| GET | `/api/indicators/{id}` | 指示器详情 |
| PUT | `/api/indicators/{id}` | 更新指示器名称/位置 |
| DELETE | `/api/indicators/{id}` | 删除指示器 |
| POST | `/api/indicators/{id}/signal` | 指示器信号上报（`triggered`/`reset`） |
| POST | `/api/indicators/{id}/suspicious` | 标记/解除可疑 |
| GET | `/api/faults` | 故障事件列表（`?status=&feederId=&longOutage=&limit=&offset=`） |
| GET | `/api/faults/{id}` | 故障事件详情 |
| POST | `/api/faults/locate` | 故障定位推理并建单 |
| POST | `/api/faults/{id}/repair` | 开始抢修 |
| POST | `/api/faults/{id}/isolate` | 隔离区段操作确认 |
| POST | `/api/faults/{id}/restore` | 复电完成 |
| POST | `/api/faults/{id}/archive` | 归档 |
| GET | `/api/outages` | 停电记录列表（`?feederId=&limit=&offset=`） |
| GET | `/api/outages/summary` | 停电统计汇总 |
| GET | `/api/overview` | 配网总览 |
| GET | `/api/audit` | 审计日志（`?limit=&offset=`） |
| POST | `/api/admin/long-outage-scan` | 手动触发长时停电扫描 |

## 九、共享枚举/常量（前后端出现位置）

| 枚举/常量 | 后端定义 | 前端定义 |
|---|---|---|
| 开关状态 SwitchStatus（closed/open） | `domain/switch.go` | `web/enums.js` `GridEnums.SwitchStatus` |
| 开关类型 SwitchType（sectionalizer/tie/feeder_outlet） | `domain/switch.go` | `web/enums.js` `GridEnums.SwitchType` |
| 指示器状态 IndicatorStatus（triggered/reset） | `domain/indicator.go` | `web/enums.js` `GridEnums.IndicatorStatus` |
| 故障事件状态 FaultStatus（located/repairing/restored/archived） | `domain/fault.go` | `web/enums.js` `GridEnums.FaultStatus` |
| 线路状态 FeederStatus（active/inactive） | `domain/feeder.go` | `web/enums.js` `GridEnums.FeederStatus` |
| 电压等级（10kV/20kV/35kV） | `config/config.go` | `web/enums.js` `GridEnums.VoltageLevels` |
| 长时停电阈值（120 分钟） | `config/config.go`、`domain/outage.go` | `web/enums.js` `GridEnums.LONG_OUTAGE_MINUTES` |

## 十、共享前端组件与 hooks

- 组件（`web/components/`）：`TopologyGraph`（拓扑管理页 + 故障页共用）、`FaultCard`（总览页 + 故障页共用）、`IndicatorTable`（指示器页 + 定位弹窗共用）。
- hooks（`web/hooks/`）：`useFeeders()`（总览页 + 拓扑页共用）、`useFaults(filter)`（故障页 + 总览页共用）、`useTopology(feederId)`（拓扑页）。

## 十一、横切关注点

1. **结构化日志**：全局 `log/slog`，HTTP 访问日志由 `middleware/logging.go` 输出 JSON 行；`LOG_LEVEL` 控制级别。
2. **操作审计**：定位、隔离、复电、归档、拓扑变更全部留痕（handler → service → `store/audit_store.go`），另含 HTTP 请求审计中间件。
3. **长时停电扫描**：默认每 10 分钟扫描超 2 小时未复电事件（`service/sweeper.go` → store → 总览页）。
4. **全局错误处理**：统一响应格式 + panic 恢复（`middleware/error_handler.go`）。
5. **trace id**：`middleware/request_id.go` 注入 `X-Request-Id` 并透传审计。
6. **安全响应头**：`nosniff`、`DENY`、`no-referrer`、`Permissions-Policy`；API 响应 `Cache-Control: no-store`。

## 十二、可复现的主链路（curl 示例）

```bash
BASE=http://127.0.0.1:8080

# 1. 拓扑创建
curl -X POST "$BASE/api/feeders" -H 'Content-Type: application/json' \
  -d '{"name":"演示线","substation":"站A","voltageLevel":"10kV"}'
curl -X POST "$BASE/api/feeders/F-001/switches" -H 'Content-Type: application/json' \
  -d '{"name":"出线开关","switchType":"feeder_outlet"}'
curl -X POST "$BASE/api/feeders/F-001/switches" -H 'Content-Type: application/json' \
  -d '{"name":"分段A","switchType":"sectionalizer"}'
curl -X POST "$BASE/api/feeders/F-001/sections" -H 'Content-Type: application/json' \
  -d '{"name":"区段1","upstreamSwitchId":"SW-001","downstreamSwitchId":"SW-002","lengthKm":1.2}'

# 2. 指示器信号上报
curl -X POST "$BASE/api/indicators/FI-001/signal" -H 'Content-Type: application/json' \
  -d '{"status":"triggered"}'
curl -X POST "$BASE/api/indicators/FI-002/signal" -H 'Content-Type: application/json' \
  -d '{"status":"reset"}'

# 3. 故障定位 → 4. 隔离 → 5. 复电 → 6. 归档
curl -X POST "$BASE/api/faults/locate" -H 'Content-Type: application/json' \
  -d '{"feederId":"F-001"}'
curl -X POST "$BASE/api/faults/FE-001/isolate" -H 'Content-Type: application/json' \
  -d '{"operator":"调度员","sectionId":"SEC-001"}'
curl -X POST "$BASE/api/faults/FE-001/restore" -H 'Content-Type: application/json' \
  -d '{"operator":"调度员","note":"复电成功"}'
curl -X POST "$BASE/api/faults/FE-001/archive" -H 'Content-Type: application/json' \
  -d '{"operator":"调度员"}'
```

## 十三、健康检查与故障排查

| 检查 | 命令 | 预期 |
|---|---|---|
| 存活 | `curl -i http://127.0.0.1:8080/healthz` | 200，`code=0` |
| 就绪 | `curl -i http://127.0.0.1:8080/readyz` | 200，`status=ready` |
| 前端 | `curl -i http://127.0.0.1:8080/` | 200，返回 HTML |
| 配置错误 | `PORT=0 go run .` | 启动失败并输出校验错误 |
| 持久化损坏 | 写入非法 JSON 到 `DATA_FILE` 后启动 | 自动备份为 `.bak`，空库启动并告警 |
| 输入校验 | `curl '/api/feeders?limit=-1'` | 400 |

## 十四、技术说明

- Go 1.23，`module example.com/grid-fault-locate-service`，纯标准库、零第三方依赖、`CGO_ENABLED=0` 可重复构建。
- 存储：内存仓储 + JSON 文件原子持久化（临时文件 → fsync → rename → 目录 fsync）；写操作由互斥锁串行化；损坏文件备份后降级空库。
- HTTP 服务器配置完整超时（ReadTimeout/ReadHeaderTimeout/WriteTimeout/IdleTimeout）并支持 SIGINT/SIGTERM 优雅关闭。
- 前端：纯原生 HTML/CSS/JS（hash 路由），`go:embed all:web` 内嵌，禁止外部 CDN。
