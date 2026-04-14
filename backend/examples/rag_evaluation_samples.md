# OfferPilot RAG 检索效果评测样例表（结课展示版）

本表用于阶段答辩展示，覆盖 Go/Java/C++/Redis/MySQL/MQ/分布式/OS/网络/算法 等主题。

建议统一评测参数：
- 检索接口：POST /api/v1/ai/rag/search
- TopK：5
- 指标：Hit@1、Hit@3、MRR、人工相关性评分（1~5）

建议记录列说明：
- 查询语句：用户自然语言问题
- 目标知识域：该问题期望命中的主题
- 期望命中标签：用于核对是否召回正确技术方向
- 参考来源仓库：理想情况下应优先召回的语料来源
- 展示要点：答辩时解释该条目价值

| 编号 | 查询语句 | 目标知识域 | 期望命中标签 | 参考来源仓库 | 展示要点 |
|---|---|---|---|---|---|
| Q01 | C++ 智能指针 shared_ptr 和 unique_ptr 区别 | C++ 基础 | cpp | cpp_interview | 展示 C++ 语料已接入并可召回 |
| Q02 | C++ 左值右值和 move 语义怎么理解 | C++ 进阶 | cpp | cpp_interview | 展示 C++ 深度主题覆盖 |
| Q03 | 手写线程池一般要考虑哪些模块 | C++ 工程 | cpp, os | cpp_interview | 展示工程实践题召回 |
| Q04 | Linux epoll 和 select 的区别及适用场景 | Linux IO | os, network | cpp_interview | 展示系统+网络交叉召回 |
| Q05 | 进程和线程的核心区别是什么 | 操作系统 | os | cpp_interview, Interview | 展示基础题高命中 |
| Q06 | TCP 三次握手四次挥手为什么要这样设计 | 计算机网络 | network | cpp_interview, Interview | 展示网络经典题命中 |
| Q07 | HTTP 和 HTTPS 的主要差异 | 计算机网络 | network | cpp_interview, Interview | 展示协议题命中 |
| Q08 | Redis 为什么快，从 IO 模型到数据结构说一下 | Redis 原理 | redis, os | java-eight-part, interview-baguwen | 展示 Redis 原理召回 |
| Q09 | Redis 缓存穿透、击穿、雪崩怎么治理 | Redis 方案 | redis | java-eight-part | 展示场景题命中 |
| Q10 | Redis 持久化 RDB 和 AOF 的区别 | Redis 持久化 | redis | java-eight-part, interview-baguwen | 展示面试高频题命中 |
| Q11 | MySQL InnoDB 索引失效的常见原因 | MySQL 索引 | mysql | interview-baguwen | 展示数据库优化题命中 |
| Q12 | MySQL 事务隔离级别和 MVCC 的关系 | MySQL 事务 | mysql | interview-baguwen, java-eight-part | 展示事务原理题命中 |
| Q13 | Kafka 怎么保证消息不丢和有序 | 消息队列 | mq, distributed | java-eight-part, interview-baguwen | 展示 MQ 核心题命中 |
| Q14 | RabbitMQ 消息堆积如何处理 | 消息队列 | mq | interview-baguwen | 展示故障处理题命中 |
| Q15 | 分布式系统里 CAP 和一致性怎么取舍 | 分布式 | distributed | java-eight-part, interview-baguwen | 展示架构题命中 |
| Q16 | 服务注册发现有哪些方案，ZK 和 Nacos 区别 | 分布式 | distributed | java-eight-part | 展示中间件题命中 |
| Q17 | Go goroutine 调度模型 GPM 是什么 | Go 并发 | golang, os | interview-baguwen | 展示 Go 专题召回 |
| Q18 | Go channel 和 mutex 在并发控制上怎么选 | Go 并发 | golang | interview-baguwen, Interview | 展示并发题命中 |
| Q19 | Java 内存模型 JMM 的可见性和有序性 | Java 并发 | java | java-eight-part | 展示 Java 八股命中 |
| Q20 | CAS 的 ABA 问题怎么解决 | Java 并发 | java | java-eight-part | 展示并发细节题命中 |
| Q21 | 设计模式里单例模式有哪些线程安全写法 | 设计模式 | cpp, java | cpp_interview, java-eight-part | 展示跨语言模式题 |
| Q22 | 快排和归并排序复杂度与稳定性对比 | 算法 | algorithm | cpp_interview, Interview | 展示算法对比题命中 |
| Q23 | 二分查找模板在什么场景容易写错 | 算法 | algorithm | Interview | 展示基础算法题命中 |
| Q24 | 死锁发生的必要条件和排查思路 | 操作系统 | os | cpp_interview, Interview | 展示排障题命中 |
| Q25 | 面试中如何回答“为什么 Redis 用跳表” | Redis 原理 | redis | interview-baguwen, java-eight-part | 展示原理问答深度 |

## 结果填写（已回填，自动评测）

自动评测汇总（TopK=5）：
- Hit@1：0.840
- Hit@3：0.880
- MRR：0.868
- 评测时间：2026-04-15T00:54:23+08:00
- 人工相关性打分口径：Hit@1=5，Hit@3=4，仅 rank=5 命中=3，未命中=2

| 编号 | Hit@1 | Hit@3 | MRR | 人工相关性(1~5) | 备注 |
|---|---|---|---|---|---|
| Q01 | true | true | 1.000 | 5 | rank=1 repo=cpp_interview |
| Q02 | true | true | 1.000 | 5 | rank=1 repo=cpp_interview |
| Q03 | true | true | 1.000 | 5 | rank=1 repo=cpp_interview |
| Q04 | true | true | 1.000 | 5 | rank=1 repo=cpp_interview |
| Q05 | true | true | 1.000 | 5 | rank=1 repo=cpp_interview |
| Q06 | true | true | 1.000 | 5 | rank=1 repo=cpp_interview |
| Q07 | true | true | 1.000 | 5 | rank=1 repo=cpp_interview |
| Q08 | true | true | 1.000 | 5 | rank=1 repo=cpp_interview |
| Q09 | true | true | 1.000 | 5 | rank=1 repo=java-eight-part |
| Q10 | true | true | 1.000 | 5 | rank=1 repo=cpp_interview |
| Q11 | true | true | 1.000 | 5 | rank=1 repo=cpp_interview |
| Q12 | true | true | 1.000 | 5 | rank=1 repo=interview-baguwen |
| Q13 | true | true | 1.000 | 5 | rank=1 repo=java-eight-part |
| Q14 | false | false | 0.000 | 2 | 无 |
| Q15 | false | false | 0.200 | 3 | rank=5 repo=java-eight-part |
| Q16 | true | true | 1.000 | 5 | rank=1 repo=java-eight-part |
| Q17 | false | false | 0.000 | 2 | 无 |
| Q18 | false | true | 0.500 | 4 | rank=2 repo=interview-baguwen |
| Q19 | true | true | 1.000 | 5 | rank=1 repo=java-eight-part |
| Q20 | true | true | 1.000 | 5 | rank=1 repo=java-eight-part |
| Q21 | true | true | 1.000 | 5 | rank=1 repo=cpp_interview |
| Q22 | true | true | 1.000 | 5 | rank=1 repo=cpp_interview |
| Q23 | true | true | 1.000 | 5 | rank=1 repo=cpp_interview |
| Q24 | true | true | 1.000 | 5 | rank=1 repo=cpp_interview |
| Q25 | true | true | 1.000 | 5 | rank=1 repo=cpp_interview |
