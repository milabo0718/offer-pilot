# RAG 检索评测结果（自动生成）

- 生成时间: 2026-04-15T00:54:23+08:00
- TopK: 5
- 样本总数: 25
- Hit@1: 0.840
- Hit@3: 0.880
- MRR: 0.868

| 编号 | 查询语句 | 期望标签 | Hit@1 | Hit@3 | MRR | 首个相关命中证据 |
|---|---|---|---|---|---|---|
| Q01 | C++ 智能指针 shared_ptr 和 unique_ptr 区别 | cpp | true | true | 1.000 | rank=1 repo=cpp_interview score=0.3332 q=:watermelon:C++11新特性 |
| Q02 | C++ 左值右值和 move 语义怎么理解 | cpp | true | true | 1.000 | rank=1 repo=cpp_interview score=0.2025 q=左值和右值 |
| Q03 | 手写线程池一般要考虑哪些模块 | cpp,os | true | true | 1.000 | rank=1 repo=cpp_interview score=0.3601 q=进程池和线程池 |
| Q04 | Linux epoll 和 select 的区别及适用场景 | os,network | true | true | 1.000 | rank=1 repo=cpp_interview score=0.2198 q=select和epoll效率 |
| Q05 | 进程和线程的核心区别是什么 | os | true | true | 1.000 | rank=1 repo=cpp_interview score=0.1738 q=进程与线程的差异 |
| Q06 | TCP 三次握手四次挥手为什么要这样设计 | network | true | true | 1.000 | rank=1 repo=cpp_interview score=0.1590 q=☆TCP四次挥手 |
| Q07 | HTTP 和 HTTPS 的主要差异 | network | true | true | 1.000 | rank=1 repo=cpp_interview score=0.2077 q=https和http区别 |
| Q08 | Redis 为什么快，从 IO 模型到数据结构说一下 | redis,os | true | true | 1.000 | rank=1 repo=cpp_interview score=0.2880 q=Redis的数据结构讲一讲 + 使用场景 |
| Q09 | Redis 缓存穿透、击穿、雪崩怎么治理 | redis | true | true | 1.000 | rank=1 repo=java-eight-part score=0.1746 q=缓存雪崩 |
| Q10 | Redis 持久化 RDB 和 AOF 的区别 | redis | true | true | 1.000 | rank=1 repo=cpp_interview score=0.1449 q=Redis持久化 |
| Q11 | MySQL InnoDB 索引失效的常见原因 | mysql | true | true | 1.000 | rank=1 repo=cpp_interview score=0.3354 q=MYSQL索引和算法原理 |
| Q12 | MySQL 事务隔离级别和 MVCC 的关系 | mysql | true | true | 1.000 | rank=1 repo=interview-baguwen score=0.2694 q=Reference |
| Q13 | Kafka 怎么保证消息不丢和有序 | mq,distributed | true | true | 1.000 | rank=1 repo=java-eight-part score=0.2670 q=认识Kafka |
| Q14 | RabbitMQ 消息堆积如何处理 | mq | false | false | 0.000 | 无 |
| Q15 | 分布式系统里 CAP 和一致性怎么取舍 | distributed | false | false | 0.200 | rank=5 repo=java-eight-part score=0.4253 q=采样 |
| Q16 | 服务注册发现有哪些方案，ZK 和 Nacos 区别 | distributed | true | true | 1.000 | rank=1 repo=java-eight-part score=0.3540 q=服务注册 |
| Q17 | Go goroutine 调度模型 GPM 是什么 | golang,os | false | false | 0.000 | 无 |
| Q18 | Go channel 和 mutex 在并发控制上怎么选 | golang | false | true | 0.500 | rank=2 repo=interview-baguwen score=0.3417 q=面试题 |
| Q19 | Java 内存模型 JMM 的可见性和有序性 | java | true | true | 1.000 | rank=1 repo=java-eight-part score=0.2942 q=3.3. 线程间通信 |
| Q20 | CAS 的 ABA 问题怎么解决 | java | true | true | 1.000 | rank=1 repo=java-eight-part score=0.2401 q=4.1. 典型 ABA 问题 |
| Q21 | 设计模式里单例模式有哪些线程安全写法 | cpp,java | true | true | 1.000 | rank=1 repo=cpp_interview score=0.2838 q=懒汉式 |
| Q22 | 快排和归并排序复杂度与稳定性对比 | algorithm | true | true | 1.000 | rank=1 repo=cpp_interview score=0.2168 q=为什么都在用快排而不是归并，堆？ |
| Q23 | 二分查找模板在什么场景容易写错 | algorithm | true | true | 1.000 | rank=1 repo=cpp_interview score=0.2665 q=**插值查找** |
| Q24 | 死锁发生的必要条件和排查思路 | os | true | true | 1.000 | rank=1 repo=cpp_interview score=0.2429 q=处理死锁 |
| Q25 | 面试中如何回答为什么 Redis 用跳表 | redis | true | true | 1.000 | rank=1 repo=cpp_interview score=0.2995 q=为何Redis使用跳表而非红黑树实现Sorted... |
