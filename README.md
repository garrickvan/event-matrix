<p align="center">
  <a href="https://eventmatrix.cn" rel="noopener">
 <img width=120px height=120px src="https://eventmatrix.cn/img/logo.svg" alt="项目logo"></a>
</p>

<h3 align="center">事件矩阵</h3>
<p>业务事件驱动，高度解耦，轻量级低代码，更适合AI时代的新型开发框架</pÏ>

<div align="center">

[![状态](https://img.shields.io/badge/status-活跃-success.svg)]()
[![Go版本](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org/doc/devel/release.html)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](https://github.com/garrickvan/event-matrix/main/LICENSE)

</div>

---

<p align="center">
    Event Matrix 是一个基于业务事件驱动的开发框架，通过高度解耦设计和轻量级低代码模式，为AI时代的系统开发提供高效灵活的解决方案。
    <br>
    <a href="https://eventmatrix.cn"><strong>探索文档 »</strong></a>
</p>

## 📝 目录

- [设计理念](#design)
- [核心特点](#features)
- [快速开始](#getting_started)
- [技术栈](#built_using)
- [贡献指南](#contributing)

## 🧐 设计理念 <a name="design"></a>

Event Matrix 秉承"简单、灵活、高效"的设计哲学，采用六边形架构与经典 MVC 模式结合，支持：

- **多架构适配**：微服务/单体/中间件架构自由选择
- **AI 协同开发**：集成 AI 辅助开发与代码生成能力
- **事件驱动开发**：业务逻辑与传输协议解耦，提升代码复用性
- **模块化设计**：即插即用的业务组件化开发模式

## ✨ 核心特点 <a name="features"></a>

- **业务事件驱动**

  - 事件定义即接口
  - 协议无关设计
  - 业务逻辑与传输层解耦

- **架构灵活性**

  - 支持微服务/单体/中间件架构
  - 平滑架构演进能力
  - 动态模块加载

- **开发效率提升**

  - 低代码开发支持
  - AI 辅助代码生成
  - 可视化规则编排

- **轻量级核心**

  - 仅需 Gateway + Worker 双角色
  - 内置服务发现与监控
  - 自动负载均衡

## 🏁 快速开始 <a name="getting_started"></a>

请参考[快速开始](https://eventmatrix.cn/docs/intro)文档。

## 🛠 技术栈 <a name="built_using"></a>

- [Hertz](https://github.com/cloudwego/hertz) - 高性能 HTTP 框架
- [Gnet](https://github.com/panjf2000/gnet) - 服务间通信
- [Gorm](https://github.com/go-gorm/gorm) - Golang ORM

## 🤝 贡献指南 <a name="contributing"></a>

我们欢迎各种形式的贡献：

1. 提交 Issue 报告问题或建议
2. Fork 仓库并提交 Pull Request
3. 完善文档或翻译
4. 参与社区讨论

## 📄 许可证

本项目采用 Apache 2.0 许可证 - 查看 [LICENSE](https://github.com/garrickvan/event-matrix/blob/main/LICENSE) 文件了解详细信息。
