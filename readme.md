# 🚀 OfferPilot (AI 模拟面试官)

> **基于多模态大模型与 RAG 的定制化 AI 模拟面试系统。**
> 项目目标是让用户先解析岗位 JD，构建岗位画像，再进入 AI 模拟面试，实现“岗位需求 -> 问答演练 -> 能力反馈”的闭环。

---

## ✨ 当前已实现能力

1. 👤 **用户系统**
   - 邮箱验证码注册
   - JWT 登录鉴权
   - 登录态路由守卫

2. 💬 **对话系统**
   - 新建会话 / 历史会话
   - 普通对话 + 流式输出（SSE）
   - RabbitMQ 异步消息入库到 MySQL

3. 📄 **JD 解析（阶段二落地）**
   - 前端新增 `JDParser` 页面
   - 调用后端 `POST /api/v1/ai/jd/parse`
   - 后端调用大模型抽取岗位结构化信息（jobTitle / skills / keywords / summary）
   - 解析成功后自动跳转到 `AIChat`
   - 前端会把 `jd_profile` 注入到后续聊天请求

4. 📚 **RAG 数据工程（阶段三核心）**
    - 多仓库题库接入：Interview + interview-baguwen + java-eight-part + cpp_interview
    - 自动清洗命令：支持 `strict/full` 两档
    - 自动结构化命令：将 Markdown 转换为问答 JSON，并自动打 `tags/difficulty`
    - 已产出可直接入库语料：
       - full：`backend/examples/rag_data_structured_full/qa_dataset.json`（990 条）
       - strict：`backend/examples/rag_data_structured_strict/qa_dataset.json`（603 条）
    - 检索评测样例表（25 条查询）：`backend/examples/rag_evaluation_samples.md`

---

## 🏗 架构概览

```text
Frontend (Vue3 + Element Plus)
        |
        | HTTP / SSE
        v
Backend API (Go + Gin + JWT)
   |            |            |
   v            v            v
MySQL        Redis       RabbitMQ
   |            |            |
   +------------+------------+
                |
                v
      AI Gateway (OpenAI-compatible / DashScope)
```

### 前端（`frontend`）

- Vue 3（Composition API）
- Vue Router + Axios
- 页面：`Login / Register / Menu / JDParser / AIChat`

### 后端（`backend`）

- Go + Gin
- 分层：`controller / service / dao / common`
- JWT 鉴权
- Gorm 持久化（MySQL）
- Redis 缓存（验证码）
- RabbitMQ 异步解耦（消息落库）

### AI 能力（`backend/common/aihelper`）

- 会话型调用：`AIHelperManager + AIHelper`
- 无状态调用（JD解析）：`ParseTextWithModel(...)`
- 支持 OpenAI 兼容协议（可配置阿里百炼兼容地址）

---

## 🧭 业务流程

### 1) JD 解析到面试

1. 用户在 `JDParser` 输入岗位描述  
2. 前端请求 `/api/v1/ai/jd/parse`  
3. 后端调用模型解析并返回结构化结果  
4. 前端保存 `jd_profile`，自动跳转 `AIChat`  
5. `AIChat` 发消息时携带 `jdProfile`，让模型回答更贴近岗位需求

### 2) 聊天落库

1. 前端发起 `/api/v1/ai/chat/*` 请求  
2. 后端生成回答（普通/流式）  
3. 用户与 AI 消息通过 RabbitMQ 异步写入 MySQL `messages`

---

## 🛠 环境依赖

- Go（建议 1.22+）
- Node.js（建议 18+）
- MySQL
- Redis Stack（需支持 RediSearch/FT.SEARCH；普通 Redis 仅验证码可用）
- RabbitMQ
- 阿里云 DashScope 百炼 API Key（使用真实 Embedding 时）

> 推荐：Windows 跑前后端 + WSL 跑中间件（也支持全本机）。

### RAG 接入后的依赖变化说明

- 相比原始项目，新增“向量检索”能力后，对 Redis 的要求从“可连接”提升为“需 RediSearch 模块可用”（建议直接用 Redis Stack）。
- 使用真实 Embedding（`text-embedding-v3`）时，需要可访问 DashScope OpenAI 兼容接口。
- 当前默认配置为真实 Embedding：`useMockEmbedding=false`，并已对齐
   - `embeddingModelName=text-embedding-v3`
   - `embeddingBaseURL=https://dashscope.aliyuncs.com/compatible-mode/v1`
   - `vectorDim=1024`
   - `batchSize=10`（接口批量上限）
- 注意：`./start-backend.sh` / `./start-backend.ps1` 会自动加载 `.env.local`；`go run ./cmd/rag_ingest ...` 这类 CLI 命令不会自动加载，需先 `export OPENAI_API_KEY=...`。

---

## 🚀 快速启动

### 1) 启动依赖服务（WSL/Linux 示例）

```bash
sudo service mysql start
sudo service redis-server start
sudo service rabbitmq-server start
```

### 2) 配置后端

编辑 `backend/config/config.yaml`：

- `mainConfig.port`：`9090`
- `mysqlConfig.host`：`127.0.0.1`（或可达地址）
- `redisConfig.host`：`127.0.0.1`（或可达地址）
- `rabbitmqConfig.host`：`127.0.0.1`（或可达地址）

### 3) 启动后端

Windows PowerShell：

```powershell
cd backend
.\start-backend.ps1
```

WSL / Linux：

```bash
cd backend
chmod +x start-backend.sh
./start-backend.sh
```

### 4) 启动前端

```bash
cd frontend
npm install
npm run serve
```

---

## 🔌 关键接口

- `POST /api/v1/ai/jd/parse`：JD 解析
- `GET /api/v1/ai/chat/sessions`：会话列表
- `POST /api/v1/ai/chat/send-new-session`：新会话问答
- `POST /api/v1/ai/chat/send`：既有会话问答
- `POST /api/v1/ai/chat/send-stream-new-session`：新会话流式
- `POST /api/v1/ai/chat/send-stream`：既有会话流式
- `POST /api/v1/ai/chat/history`：会话历史

### RAG 底座接口（新增）

- `GET /api/v1/ai/rag/health`：检查 Redis 可达性、RediSearch 可用性、索引状态
- `POST /api/v1/ai/rag/index/init`：初始化 Redis 向量索引（幂等）
- `POST /api/v1/ai/rag/ingest`：触发题库入库（支持空请求体走默认目录）
- `POST /api/v1/ai/rag/search`：向量检索（支持 `source_file` / `tags` 过滤）

检索请求示例：

```json
{
   "query": "Golang 并发 channel 和 mutex 的区别",
   "topK": 3,
   "filter": {
      "source_file": "sample_golang_backend.md",
      "tags": []
   }
}
```

---

## 🧪 RAG 最小演示

### 1) 准备示例数据

仓库已提供两份示例题库：

- `backend/examples/rag_data/sample_java_backend.md`
- `backend/examples/rag_data/sample_golang_backend.md`

扩展后的清洗与结构化数据目录：

- `backend/examples/rag_data_cleaned_full`
- `backend/examples/rag_data_cleaned`
- `backend/examples/rag_data_structured_full`
- `backend/examples/rag_data_structured_strict`

### 2) 配置项说明（`backend/config/config.yaml`）

`ragConfig` 关键字段：

- `enabled`：是否启用 RAG（当 Redis 不可用时可临时关闭）
- `chatAugmentEnabled`：是否给聊天自动注入 RAG 召回上下文
- `chatAugmentTopK`：聊天增强时的默认召回条数
- `redisAddr`：Redis 地址，例如 `127.0.0.1:6379`
- `indexName`：向量索引名
- `vectorDim`：向量维度（`text-embedding-v3` 下为 `1024`）
- `embeddingAPIKey`：真实 Embedding API Key
- `defaultTopK`：默认召回条数
- `batchSize`：向量化批大小（`text-embedding-v3` 不可大于 `10`）
- `useMockEmbedding`：是否启用 Mock（真实环境建议 `false`）

### 3) 执行入库（CLI）

```bash
cd backend
export OPENAI_API_KEY="你的DashScopeKey"
go run ./cmd/rag_ingest -dir ./examples/rag_data -mock=false
```

使用结构化问答集入库（推荐展示用）：

```bash
cd backend
export OPENAI_API_KEY="你的DashScopeKey"
go run ./cmd/rag_ingest -dir ./examples/rag_data_structured_strict -mock=false
```

使用全量问答集入库（追求覆盖面）：

```bash
cd backend
export OPENAI_API_KEY="你的DashScopeKey"
go run ./cmd/rag_ingest -dir ./examples/rag_data_structured_full -mock=false
```

### 4) 数据清洗与结构化命令（可复现）

清洗（strict）：

```bash
cd backend
go run ./cmd/rag_prepare -profile strict -out ./examples/rag_data_cleaned -min-runes 120
```

清洗（full）：

```bash
cd backend
go run ./cmd/rag_prepare -profile full -out ./examples/rag_data_cleaned_full -min-runes 50
```

二次结构化（strict）：

```bash
cd backend
go run ./cmd/rag_structify -in ./examples/rag_data_cleaned -out ./examples/rag_data_structured_strict -min-answer-runes 80
```

二次结构化（full）：

```bash
cd backend
go run ./cmd/rag_structify -in ./examples/rag_data_cleaned_full -out ./examples/rag_data_structured_full -min-answer-runes 80
```

统计报告位置：

- `backend/examples/rag_data_structured_full/_meta/structify_report.json`
- `backend/examples/rag_data_structured_strict/_meta/structify_report.json`
- `backend/examples/rag_evaluation_samples.md`

自动评测（25条查询，输出 Hit@1/Hit@3/MRR 与命中证据）：

```bash
cd backend
go run ./cmd/rag_eval -topk 5
```

评测输出文件：

- `backend/examples/rag_evaluation_results.md`
- `backend/examples/_meta/rag_evaluation_results.json`

最新一次自动评测结果（25条查询，TopK=5）：

- 评测时间：2026-04-15T00:54:23+08:00
- Hit@1：0.840
- Hit@3：0.880
- MRR：0.868
- 已回填展示表：`backend/examples/rag_evaluation_samples.md`

如果本地 Redis 不支持 `FT.CREATE` / `FT.SEARCH`，请改用 Redis Stack。

### 4) 一键最小闭环演示（索引 + 入库 + 检索）

```bash
cd backend
go run ./cmd/rag_demo
```

该命令会打印：

- 入库统计（总 chunk、成功、失败）
- 使用 JD 关键词检索到的 topK 结果（含 score、source、section、content）

---

## 🔧 排障指南

1. 前端 404/500：检查 `frontend/vue.config.js` 代理是否指向可达后端（建议 `127.0.0.1:9090`）。
2. 后端启动即退出：优先看 Redis/RabbitMQ 地址和端口。
3. JD 解析成功但看不到结果：  
   - 浏览器 Network 看 `/api/v1/ai/jd/parse` 响应  
   - 或控制台读取 `localStorage.getItem("jd_profile")`
4. 流式失败：确认请求路径顺序为 `/api/v1/ai/...`（不是 `/api/ai/v1/...`）。

---

## 📍 Roadmap

- [x] 基础框架与用户系统
- [x] 聊天（普通+流式）与异步持久化
- [x] JD 解析接口 + 前端入口 + 自动跳转面试
- [x] RAG 检索增强（阶段三）
- [x] RAG 数据工程（多仓库清洗 + 问答结构化 + 评测样例）
- [ ] STT / TTS 语音链路（阶段四）
- [ ] 异步评分与雷达图（阶段五）

---

## 🤝 协作约定

- `.env.local` 不入仓库
- 真实密钥不提交
- 提交前至少验证：登录 -> JD解析 -> AIChat 一条完整链路
