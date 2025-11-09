# Kyogre - 数据库压测工具

<div align="center">

![Kyogre](https://img.pokemondb.net/sprites/home/normal/kyogre.png)

**像盖欧卡引发暴雨洪水一样，模拟海量数据流量的数据库压测工具**

[![Go Version](https://img.shields.io/badge/Go-1.19+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

</div>

## 🚀 项目简介

Kyogre 是一个高性能、多数据库支持的压测工具，以宝可梦中的盖欧卡(Kyogre)命名，象征着它能够像引发暴雨洪水一样，模拟海量数据流量对多种数据库进行全方位的压力测试。

## ✨ 核心特性

### 🗄️ 多数据库支持
- **MySQL**: 完整的 SQL 操作支持
- **MongoDB**: 文档数据库压测
- **Redis**: 内存数据库压测
- **扩展性**: 易于添加新的数据库类型

### 📊 压测模式
- **一次性数据**: 预定义数据量的压测
- **持续性数据**: 持续生成流量的压测
- **混合模式**: 多种操作类型的组合压测

### 🔧 MySQL 深度支持
- **DML操作**: 增、删、改、查
- **DDL操作**: 表结构变更
- **大事务**: 复杂事务场景模拟
- **Ghost操作**: 在线DDL变更压测
- **连接池测试**: 数据库连接压力测试

### 📈 监控指标
- 吞吐量(QPS/TPS)
- 响应时间分布
- 错误率和异常统计
- 资源使用情况
- 实时性能图表

## 🛠️ 快速开始


报告内容包括：
- 📊 吞吐量趋势图
- ⏱️ 响应时间分布
- 🔴 错误统计和分析
- 📉 资源使用情况
- 🎯 性能瓶颈识别

## 🏗️ 架构设计

---

<div align="center">

**像盖欧卡掌控海洋一样，Kyogre 助你掌控数据库性能！**

</div>