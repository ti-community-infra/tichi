# ti-community-owners

## 设计背景

设计 owners 主要是为 ti-community-lgtm 和 ti-community-merge 服务，在 Kubernetes 社区中，他们使用 [OWNERS](https://github.com/kubernetes/test-infra/blob/master/OWNERS) 文件定义权限，在该文件中指定了当前目录及子目录的 reviewers 和 approvers。但是目前 TiDB 社区已经依赖于社区先前的 [Bot](https://github.com/pingcap-incubator/cherry-bot) 运行了很长一段时间。大家在先前的 Bot 的使用过程中摸索出了一套协作机制。如果直接采用 Kubernetes 社区的机制会带来非常高的学习成本，水土不服。

所以我们决定研发一个适合当前 TiDB 社区协作模式的权限控制服务，基于目前 TiDB 社区的 [SIG](https://github.com/pingcap/community) 架构来定义每个 PR 的权限。

## 权限设计

基于目前 TiDB 的 SIG 的设计，将权限划分如下：

- committers（**可以合并 PR**）
  - maintainers
  - techLeaders
  - coLeaders
  - committers
- reviewers (**可以赞同 PR**)
  - maintainers
  - techLeaders
  - coLeaders
  - committers
  - reviewers

在 TiDB 社区的协作过程中，**一个 PR 一般要经过多次 review 之后才能进行合并**。所以在这个服务中要定义清楚每个 PR 需要的 `LGTM` 个数。也可以在配置中指定 PR 要求 `LGTM` 个数的 label 前缀。owners 会自动从 label 中读取要求的 `LGTM` 个数。

## 设计思路

为了实现以上描述的权限划分，我们决定采取 RESTFUL 接口来定义每个 PR 的权限。

接口路径：`/repos/:org/:repo/pulls/:number/owners`

因为要基于 SIG 来划分权限，所以要求这些 PR 中能够获取到当前 PR 所属的 SIG。owners 会在当前 PR 中查找以 `sig/` 开头的标签，然后查找该 SIG 的信息。最终根据获取到的 SIG 的信息生成 owners。

但是确实存在一些特殊情况找不到对应的 SIG：
- 一些模块暂时未划分清楚 SIG 所属：使用 TiDB 社区所有 SIG 的 reviewers 和 committers
- 一些小型仓库直接隶属于某个 SIG: 支持为该仓库配置默认的 SIG
- 一些仓库的 PR 跟 SIG 无关：支持使用仓库的 [GitHub 权限](https://docs.github.com/en/organizations/managing-access-to-your-organizations-repositories/repository-permission-levels-for-an-organization)
  - committers（**可以合并 PR**）
    - Admin 权限
    - Maintain 权限
    - Write 权限
  - reviewers (**可以赞同 PR**)
    - Triage 权限

注：因为 maintainers 没有隶属于任何一个 SIG，所以会通过一个配置项来直接从 GitHub team 获取。

## 参数配置

| 参数名                     | 类型                    | 说明                                                                       |
|----------------------------|-------------------------|----------------------------------------------------------------------------|
| repos                      | []string                | 配置生效仓库                                                               |
| sig_endpoint               | string                  | 获取 SIG 信息的 RESTFUL 接口地址                                           |
| default_sig_name           | string                  | 为该仓库设置默认的 SIG                                                     |
| default_require_lgtm       | int                     | 为该仓库设置默认需要的 lgtm 个数                                           |
| require_lgtm_label_prefix  | string                  | 插件支持通过标签指定当前 PR 需要的 lgtm 个数，该选项用于设置相关标签的前缀 |
| trusted_teams (deprecated) | []string                | 信任的 GitHub team 名称列表（一般为 maintainers team）                     |
| use_github_teams           | bool                    | 通过 GitHub Teams 来获取 Committers 和 Reviewers 列表                      |
| committer_teams            | []string                | 指定其成员可以作为 Committer 的 GitHub Team 名称列表                       |
| reviewer_teams             | []string                | 指定其成员可以作为 Reviewer 的 GitHub Team 名称列表                        |
| use_github_permission      | bool                    | 使用 GitHub 权限                                                           |
| branches                   | map[string]BranchConfig | 分支粒度的参数配置, map结构的key是分支名称，对分支的配置会覆盖对仓库的配置 |

### BranchConfig

| 参数名                     | 类型     | 说明                                                         |
|----------------------------|----------|--------------------------------------------------------------|
| default_require_lgtm       | int      | 为该分支设置默认需要的 lgtm 个数                             |
| trusted_teams (deprecated) | []string | 为该分支设置信任的 GitHub team                               |
| use_github_teams           | bool     | 通过 GitHub Teams 来获取 Committers 和 Reviewers 列表        |
| committer_teams            | []string | 为该分支设置其成员可以作为 Committer 的 GitHub Team 名称列表 |
| reviewer_teams             | []string | 为该分支设置其成员可以作为 Reviewer 的 GitHub Team 名称列表  |
| use_github_permission      | bool     | 使用 GitHub 权限                                             |

例如：

```yml
ti-community-owners:
  - repos:
      - ti-community-infra/test-live
      - ti-community-infra/tichi
      - ti-community-infra/ti-community-bot
      - ti-community-infra/ti-challenge-bot
    sig_endpoint: https://bots.tidb.io/ti-community-bot
    require_lgtm_label_prefix: require/LGT
    use_github_teams: true
    committer_teams:
      - bots-committers
    reviewer_teams:
      - bots-reviewers
    branches:
      release:
        default_require_lgtm: 2
        use_github_permission: true
```

## Q&A

### 如何查看当前 PR 的权限？

直接通过与 GitHub 一致的 RESTFUL 接口查看，例如：[ti-community-infra/test-dev/pulls/179](https://prow.tidb.io/ti-community-owners/repos/ti-community-infra/test-dev/pulls/179/owners)
