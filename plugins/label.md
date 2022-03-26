# ti-community-label

## 设计背景 

在 TiDB 社区中，Issue 和 PR 都有大量的标签，我们在使用过程中比较混乱，很多 label 定义和分类的也不够清楚。原来的机器人只支持 `/label` 这样一个打标签的命令。这导致在使用该命令过程中很痛苦，因为很多时候标签会有分类的前缀，标签会比较长，使用时老是记不清楚标签。

ti-community-label 采取不一样的策略，该插件支持按照分类打标签。例如针对 `type` 这一类的 label，我们就可以使用 `/[remove-]type bug` 这样的命令，更加语义化的为 Issue 或者 PR 打上或取消 `type/bug` 的标签。除此之外，我们也保留了 `/[remove-]label` 命令来打上或取消一些无法分类的标签。

## 权限设计

该插件主要负责的是为 Issue 或者 PR 添加 label，所以我们将权限设置的为GitHub 用户都可以使用该功能。

## 设计思路

该插件主要参考了 Kubernetes 的 label 插件设计，在它的基础上扩展，支持为每个仓库自定义 label 前缀（也就是分类）。这样在使用过程中，大家就可以为仓库分类整理标签。

除此之外，在实现该插件的过程中要注意：**因为该插件权限设置的比较宽松，所以只能添加该仓库已经创建好的 label。不然有可能导致被打上一些无用的标签。**

## 参数配置

| 参数名            | 类型     | 说明                                                                          |
| ----------------- | -------- | ----------------------------------------------------------------------------- |
| repos             | []string | 配置生效仓库                                                                  |
| additional_labels | []string | 无法分类的 labels                                                             |
| prefixes          | []string | 分类前缀                                                                      |
| exclude_labels    | []string | 一些不希望被该插件添加或移除的 labels （例如：一些只允许机器人操作的 labels） |

例如：

```yml
ti-community-label:
  - repos:
      - ti-community-infra/test-live
      - ti-community-infra/tichi
      - ti-community-infra/prow-configs
      - ti-community-infra/ti-community-bot
      - ti-community-infra/ti-challenge-bot
    prefixes:
      - type
      - status
    additional_labels:
      - 'wontfix'
      - 'duplicate'
    exclude_labels:
      - 'status/can-merge'
```

**如果需要添加 `help wanted` 或 `good first issue` 标签，请使用 [help](https://prow.tidb.io/command-help#help) 插件提供的 `/[remove-]help` 和 `/[remove-]good-first-issue` 命令。**

## 参考文档

- [command help](https://prow.tidb.io/command-help?repo=ti-community-infra%2Ftichi#type)
- [代码实现](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/label)

## Q&A

### 为什么使用该功能添加标签没有反应？

请检查该仓库是否存在该 label，插件只会添加仓库已经创建的标签。