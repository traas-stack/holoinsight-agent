/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package executor

//v1版本:
//1. 不考虑任何合并, 每个采集任务的执行保持完全独立
//2. 不考虑任何抽象:可以没必要用接口, 随意耦合, 直接依赖实现类
//3. 不考虑任何流程统一/复用: 即明显不同的流水线可以分开处理
//4. 有如下角色
//4.1 PullLogSource 基于拉模式的日志数据源
//4.2 LogConsumer 日志消费者, 先不用管它实现了什么功能(聚合还是产出明细数据)
//4.3 PullLogPipeline 协调一个 PullLogSource 和 LogConsumer
//4.4 SelfDriverSource 自我驱动, 自动产出结构化数据的数据源
//4.5 XxxLogConsumer 结构化日志消费者?
//4.6 XxxPipeline
//5. 一些细节

// v2版本:
// 1. 合并数据源, 数据源产生一次, 消费多次, 共享同一个数据源的消费者执行可能有某些限制(比如所有消费者是串行处理的, 一个处理完再换下一个, 而不是各有个的goroutine)
// 1.1 此时引入 ConsumerManager 角色来协调多个Consumer, 多个Consumer应该实现相同的接口, 或者让 ConsumerManager 使用 switch/case 的方式兼容多种 Consumer
// 2. 不考虑任何抽象:可以没必要用接口, 随意耦合, 直接依赖实现类
// 3. 不考虑任何流程统一/复用: 即明显不同的流水线可以分开处理
