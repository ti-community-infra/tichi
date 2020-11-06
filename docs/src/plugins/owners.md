# ti-community-owners

## 设计背景

设计 owners 主要是为了 ti-community-lgtm 和 ti-community-merge 服务，在 Kubernetes 社区中，他们使用 [OWNERS](https://github.com/kubernetes/test-infra/blob/master/OWNERS) 文件定义权限，在该文件中指定了当前目录及子目录的 reviewers 和 approvers。但是目前 TiDB 社区已经依赖于社区先前的 [Bot](https://github.com/pingcap-incubator/cherry-bot) 运行了很长一段时间。大家在先前的 Bot 的使用过程中摸索出了一套协作机制。如果直接采用 Kubernetes 社区的机制会带来非常高的学习成本，水土不服。

所以决定研发一个适合当前 TiDB 社区协作模式的权限控制服务，基于目前 TiDB 社区的 [sig](https://github.com/pingcap/community) 架构定义了每个 PR 的权限。

## 权限设计

基于目前 TiDB 的 sig 的设计，将权限划分如下：

- approvers（**可以使用 /merge 命令**）
  - maintainers
  - techLeaders
  - coLeaders
  - committers
- reviewers (**可以使用 /lgtm 命令**)
  - maintainers
  - techLeaders
  - coLeaders
  - committers
  - reviewers

在 TiDB 相关社区的协作过程中，**一个 PR 一般要经过多次 review 之后才能进行合并**。所以在这个服务中该要定义清楚每个 PR 需要的 `LGTM` 个数。

## 实现思路

为了实现以上描述的权限划分，最终决定采取 RESTFUL 接口来定义每个 PR 的权限。

接口路径：`/repos/:org/:repo/pulls/:number/owners`

因为要基于 sig 来划分权限，所以要求这些 PR 中能够获取到当前 PR 所属的 sig。owners 会在当前 PR 中查找以 `sig/` 开头的标签，然后查找该 sig 的信息。最终根据获取到的实时的 sig 的信息生成 owners。

但是可能确实存在一些特殊情况找不到对应的sig：
- 一些模块暂时未划分清楚 sig：使用当前仓库的 collaborator
- 一些小型仓库直接隶属于某个 sig: 支持为该仓库配置默认的 sig

这样基本上就能够实现该服务。

注：因为 maintainers 没有隶属于任何一个 sig，所以会通过一个配置项来直接从 GitHub team 获取。

## 参数配置

| 参数名                  | 类型     | 说明                                          |
| ----------------------- | -------- | --------------------------------------------- |
| repos                   | []string | 配置生效仓库                                  |
| sig_endpoint            | string   | 获取 sig 信息 RESTFUL 接口地址                |
| default_sig_name        | string   | 为该仓库设置默认 sig 名字                     |
| trusted_team_for_owners | string   | 信任的 GitHub team（一般为 maintainers team） |

例如：

```yml
ti-community-owners:
  - repos:
      - tidb-community-bots/test-live
      - tidb-community-bots/ti-community-prow
      - tidb-community-bots/ti-community-bot
      - tidb-community-bots/ti-challenge-bot
    sig_endpoint: https://bots.tidb.io/ti-community-bot
```

## Q&A

### 如何查看当前 PR 的权限？

直接通过与 GitHub 一致的 RESTFUL 接口查看，例如：[tidb-community-bots/test-dev/pulls/179](https://prow.tidb.io/ti-community-owners/repos/tidb-community-bots/test-dev/pulls/179/owners)