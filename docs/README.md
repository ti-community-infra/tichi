# Prow 介绍

## 关于 Prow

[Prow](https://github.com/kubernetes/test-infra/tree/master/prow) 是一个基于 Kubernetes 的 CI/CD 平台。CI 任务可以被各种类型的事件触发并且反馈状态给不同的服务。除了执行任务，Prow 通过 `/foo` 风格命令以及自动PR合并的策略提供 GitHub 自动化。

## 在 TiDB 社区的应用

在 [tichi](https://github.com/ti-community-infra/tichi) 中，主要利用 Prow 提供的 GitHub 自动化功能，希望实现 [TiDB](https://github.com/pingcap/tidb) 社区的协作流程高度自动化和标准化。

因为 Prow 具有良好的[扩展性](https://github.com/kubernetes/test-infra/tree/master/prow/plugins)，可以按照自己的社区实践规范编写和定制插件。TiDB 社区基于这个特性定制了一批自己的[插件](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins)。

## 关于本书

编写本书的目的：一方面是希望本书作为一本手册能够让 TiDB 相关社区的协作者查阅，另外一方面也希望能够作为一个使用 Prow 的例子供其他社区参考。

