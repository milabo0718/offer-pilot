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
- Redis
- RabbitMQ

> 推荐：Windows 跑前后端 + WSL 跑中间件（也支持全本机）。

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
- [ ] RAG 检索增强（阶段三）
- [ ] STT / TTS 语音链路（阶段四）
- [ ] 异步评分与雷达图（阶段五）

---

## 🤝 协作约定

- `.env.local` 不入仓库
- 真实密钥不提交
- 提交前至少验证：登录 -> JD解析 -> AIChat 一条完整链路
