# Prow 介绍

## 关于 Prow

[Prow](https://github.com/kubernetes/test-infra/tree/master/prow) 是一个基于 Kubernetes 的 CI/CD 平台。CI 任务可以被各种类型的事件触发并且反馈状态给不同的服务。除了执行任务，Prow 通过 `/foo` 风格命令以及自动PR合并的策略提供 GitHub 自动化。

## 在 TiDB 社区的应用

在 [tichi](https://github.com/ti-community-infra/tichi) 中，主要利用 Prow 提供的 GitHub 自动化功能，希望实现 [TiDB](https://github.com/pingcap/tidb) 社区的协作流程高度自动化和标准化。

因为 Prow 具有良好的[扩展性](https://github.com/kubernetes/test-infra/tree/master/prow/plugins)，可以按照自己的社区实践规范编写和定制插件。TiDB 社区基于这个特性定制了一批自己的[插件](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins)。

## 常用插件列表

以下展示的是 TiDB 社区当中较为常用的一些组件或插件，其中名称以 `ti-community-` 开头的外部插件是 TiDB 社区 [Community Infra SIG](https://developer.tidb.io/SIG/community-infra) 开发并正在维护的插件。如果你在使用过程中遇到任何问题，你都可以在 [Slack](https://slack.tidb.io/invite?team=tidb-community&channel=sig-community-infra&ref=github) 当中与我们取得联系。

| 插件名称                        | 插件类型 | 功能简介                                                                                      |
| ------------------------------- | -------- | --------------------------------------------------------------------------------------------- |
| tide                            | 基础组件 | 通过一些给定条件来管理 GitHub PR 池，自动合并符合条件的 PR。                                  |
| ti-community-owners             | 外部插件 | 根据 SIG 或 Github 权限等信息来确定 PR 的 reviewer 和 committer。                             |
| ti-community-lgtm               | 外部插件 | 添加命令添加或取消 `status/LGT*` 标签。                                                       |
| ti-community-merge              | 外部插件 | 通过命令添加或删除 PR 的 `status/can-merge` 标签。                                            |
| ti-community-blunderbuss        | 外部插件 | 主要负责基于 SIG 或 Github 权限自动分配 reviewer。                                            |
| ti-community-autoresponder      | 外部插件 | 根据评论内容自动回复。                                                                        |
| ti-community-tars               | 外部插件 | 主要负责自动将主分支合并到当前 PR，以确保当前 PR 的 Base 保持最新。                           |
| ti-community-label              | 外部插件 | 通过命令为 PR 或 Issue 添加标签。                                                             |
| ti-community-label-blocker      | 外部插件 | 主要负责阻止用户对某些敏感标签的进行非法操作。                                                |
| ti-community-label-contribution | 外部插件 | 主要负责为外部贡献者的 PR 添加 `contribution` 或 `first-time-contributor` 标签。              |
| need-rebase                     | 外部插件 | 当 PR 需要进行 rebase 时，通过添加标签或添加评论提醒 PR 作者进行 rebase。                     |
| require-matching-label          | 内置插件 | 当 PR 或 Issue 缺失相关标签时，通过添加标签或评论提醒贡献者进行补充。                         |
| hold                            | 内置插件 | 通过 `/[un]hold` 命令，添加或取消 PR 的不可合并状态。                                         |
| assign                          | 内置插件 | 通过 `/[un]assign` 命令，添加或取消 PR 或 Issue 的 assignee。                                 |
| size                            | 内置插件 | 根据代码修改行数评估 PR 的大小，并为 PR 打上 `size/*` 标签。                                  |
| lifecycle                       | 内置插件 | 通过标签标记 Issue 或 PR 的生命周期。                                                         |
| wip                             | 内置插件 | 为正在开发的 PR 添加 `do-not-merge/work-in-progress` 标签，阻止自动分配 reviewer 和 PR 合并。 |
| welcome                         | 内置插件 | 通过机器人向首次贡献的贡献者发送欢迎语。                                                      |
| label_sync                      | 工具     | 能够将 yaml 文件当中配置的标签同步到一个或多个仓库。                                          |
| autobump                        | 工具     | 通过自动提交 Pull Request 的方式更新上游 Prow 及其相关组件和插件的版本。                      |

同时，你可以在 [Plugins](https://prow.tidb.io/plugins) 当中找到目前所有可用的组件或插件，也可以在 [Command](https://prow.tidb.io/command-help) 当中查看指定仓库可用的命令。

如果你想通过 tichi 实现一个新的功能，你可以通过 [RFC](https://github.com/ti-community-infra/rfcs) 方式提出需求，以便能够在社区当中展开广泛的沟通和交流，从而最终确定新功能的具体需求和实施方案。 

## 关于本书

编写本书的目的：一方面是希望本书作为一本手册能够让 TiDB 相关社区的协作者查阅，另外一方面也希望能够作为一个使用 Prow 的例子供其他社区参考。
