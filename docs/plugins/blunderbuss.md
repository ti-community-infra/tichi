# ti-community-blunderbuss

## 设计背景

在 TiDB 社区中，因为一个 PR 会经过多阶段 review，所有我们希望能够在 PR 被创建的时候自动分配 reviewers。

ti-community-blunderbuss 负责在 PR 创建时，根据 ti-community-owners 划分的权限自动分配 reviewers。除此之外，我们需要考虑如果 reviewers 长时间无回复时需要再次请求其他人 review 的情况，所以我们支持了 `/auto-cc` 命令来触发再次分配 reviewers。

实际上在一些 TiDB 社区的仓库当中，绝大多数 PR 都需要带有 sig 标签，只有在添加了 sig 标签之后才能够自动分配 reviewers，所以我们需要通过配置加以限制，减少不必要的自动分配。

## 权限设计

该插件主要负责 reviewers 的自动分配，所以我们将权限设置为 GitHub 用户都可以使用该功能。

## 设计思路

该插件主要参考了 Kubernetes 的 blunderbuss 插件设计。在它的基础上，我们依托于 ti-community-owners 实现 PR 的 reviewers 自动分配。

插件的分配策略为：

- 如果有权限的 reviewers 数量小于或等于 `max_request_count`
  - 分配所有有权限的 reviewers
- 如果有权限的 reviewers 数量大于 `max_request_count`
  - 获取 PR 的所有文件改动，找出这些改动文件的历史贡献者，并根据贡献者对文件的改动次数计算得到权重来进行加权随机分配

如果一个仓库要求 PR 带有 sig 标签才能进行自动分配，那么在 PR 被添加上 sig 相关标签之前，创建 PR、使用 `/auto-cc` 命令都不会进行自动分配。当我们添加 sig 标签之后，插件才会自动的分配 reviewers。

**需要特别注意的是**：当 PR 的 Body 中使用了 `/cc` 命令或者已经手动指定了 reviewers 之后，插件不会再进行自动分配。但是使用 `/auto-cc` 命令无该限制。

## 参数配置

| 参数名                | 类型     | 说明                                                                       |
| --------------------- | -------- | -------------------------------------------------------------------------- |
| repos                 | []string | 配置生效仓库                                                               |
| pull_owners_endpoint  | string   | PR owners RESTFUL 接口地址                                                 |
| max_request_count     | int      | 最多的分配人数（不配置将分配所有 reviewers）                               |
| include_reviewers     | []string | 只有这些 reviewers 参与自动分配（针对一些存在大量不活跃 reviewers 的仓库） |
| exclude_reviewers     | []string | 不参与自动分配的 reviewers（针对一些可能不活跃的 reviewers ）              |
| grace_period_duration | int      | 配置等待其它插件添加 sig 标签的等待时间，单位为秒，默认为 5 秒             |
| require_sig_label     | bool     | PR 是否必须带有 SIG 标签才允许自动分配 reviewers                           |

例如：

```yml
ti-community-blunderbuss:
  - repos:
      - ti-community-infra/test-live
    pull_owners_endpoint: https://prow-dev.tidb.net/ti-community-owners
    max_request_count: 1
    exclude_reviewers:
      # 机器人
      - ti-chi-bot
      - mini-bot
      # 不活跃的 reviewers
      - sykp241095
      - AndreMouche
    grace_period_duration: 5
    require_sig_label: true
```

## 参考文档

- [command help](https://prow.tidb.net/command-help?repo=ti-community-infra%2Fconfigs#auto_cc)
- [代码实现](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/blunderbuss)

## Q&A

### 为什么使用 /auto-cc 命令不会自动分配 reviewers？

可能是因为你的仓库设置了 `require_sig_label`，导致在添加 sig 标签之前都不会自动分配 reviewers。
