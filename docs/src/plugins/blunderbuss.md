# ti-community-blunderbuss

## 设计背景

在 TiDB 社区中，因为一个 PR 会经过多人多阶段 review，所有我们希望能够在 PR 被创建的时候自动分配 reviewers。

ti-community-blunderbuss 负责在 PR 创建时，根据 ti-community-owners 划分的权限自动分配 reviewers。除此之外，我们还需要考虑到如果 reviewers 长时间无回复时需要再次请求其他人 review  的情况，所以我们还支持了 `/auto-cc` 命令来触发再次分配 reviewers。

## 权限设计

该插件主要负责 reviewers 的自动分配，所以我们将权限设置为 GitHub 用户都可以使用该功能。

## 设计思路

该插件主要参考了 Kubernetes 的 blunderbuss 插件设计。在它的基础上，我们依托于 ti-community-owners 实现当前 PR 的 reviewers 自动分配。

## 参数配置

| 参数名               | 类型     | 说明                                                        |
| -------------------- | -------- | ----------------------------------------------------------- |
| repos                | []string | 配置生效仓库                                                |
| pull_owners_endpoint | string   | PR owners RESTFUL 接口地址                                  |
| max_request_count    | int      | 最多的分配人数                                              |
| exclude_reviewers    | []string | 不参与自动分配的 reviewers（针对一些可能不活跃的 reviewers ）   |
| grace_period_duration| int      | 配置等待其它插件添加 sig 标签的等待时间，单位为秒，默认为 5 秒    |

例如：

```yml
ti-community-blunderbuss:
  - repos:
      - tidb-community-bots/test-live
    pull_owners_endpoint: https://prow-dev.tidb.io/ti-community-owners
    max_request_count: 1
    exclude_reviewers:
      # Bots
      - ti-community-prow-bot
      - rustin-bot
      # Inactive reviewers
      - sykp241095
      - AndreMouche
    grace_period_duration: 5
```

## 参考文档

- [command help](https://prow.tidb.io/command-help?repo=tidb-community-bots%2Fconfigs#auto_cc)
- [代码实现](https://github.com/tidb-community-bots/ti-community-prow/tree/master/internal/pkg/externalplugins/blunderbuss)

## Q&A

### 如果我的 PR 更改了 sig 的 label 导致了 owners 中的 reviewers 发生了变动怎么办？

目前我们还未做这部分支持，我们后续会根据 label 的变化自动更换 reviewers 的请求。
