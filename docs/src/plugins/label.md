# ti-community-label

## 设计背景 

在 TiDB 社区中，issues 和 PRs 都有大量的标签，我们在使用过程中比较混乱，很多 label 定义和分类的也不够清楚。目前的机器人只支持 `/label` 这样一个打标签的命令。这导致在使用该命令过程中很痛苦，因为很多时候标签会有分类的前缀，标签会比较长，使用时老是记不清楚标签。

ti-community-label 采取不一样的策略，该插件支持按照分类打标签。例如针对 `type` 这一类的 label，我们就可以使用 `/[remove-]type bug` 这样的命令，更加语义化的为 issue 或者 PR 打上或取消 `type/bug` 的标签。除此之外，我们也保留了 `/[remove-]label` 命令来打上或取消一些无法分类的标签。

## 权限设计

该插件主要负责的是为 issue 或者 PR 添加 label，所以我们将权限设置的比较宽松。GitHub 用户都可以使用该功能。

## 实现思路

该插件主要参考了 Kubernetes 的 label 插件设计，在它的基础上扩展，支持了为每个仓库自定义 label 前缀也就是分类。这样在使用过程中，大家就可以在自己的仓库分类整理自己的标签。

除此之外，在实现该插件的过程中要注意：**因为该插件权限设置的比较宽松，所以只能添加该仓库已经创建好的 label。不然有可能导致被恶意操作或者打上一些无用的标签。**

## 参数配置

| 参数名           | 类型     | 说明                    |
| ---------------- | -------- | ----------------------- |
| repos            | []string | 配置生效仓库            |
| AdditionalLabels | []string | 额外的无法分类的 labels |
| Prefixes         | []string | 分类前缀                |

例如：

```yml
ti-community-label:
  - repos:
      - tidb-community-bots/test-live
      - tidb-community-bots/ti-community-prow
      - tidb-community-bots/ti-community-bot
      - tidb-community-bots/ti-challenge-bot
    prefixes:
      - type
```

## 参考文档

- [command help](https://prow.tidb.io/command-help?repo=tidb-community-bots%2Fti-community-prow#type)
- [代码实现](https://github.com/tidb-community-bots/ti-community-prow/tree/master/internal/pkg/externalplugins/merge)

## Q&A

### 为什么我使用该功能添加标签没有反应？

请检查该仓库是否存在该 label，插件只会添加仓库已经创建的标签。