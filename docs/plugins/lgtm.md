# ti-community-lgtm

## 设计背景

在 TiDB 社区中，我们采用了多阶段 code review 的方式来进行协作。一个 PR 一般会经过多个人的 review，才能达到合并的基本条件。例如：当 PR 被第一个人 review 之后，会为该 PR 打上 `status/LGT1` 的标签。然后当 PR 被第二个 review 之后，会为该 PR 打上 `status/LGT2` 的标签。每个 sig 都会设置默认需要的 LGTM 的个数，一般为 2 个。

ti-community-lgtm 是会根据命令和权限自动的为 PR 添加 LGTM 对应 label 的插件。它会作为一个独立的服务部署，由 Prow Hook 将 GitHub 的 webhook 事件转发给该插件进行处理。

## 权限设计

该插件主要负责 code review 的协作过程，权限如下：

- `/lgtm` 或 GitHub Approve
  - reviewers
    - maintainers
    - techLeaders
    - coLeaders
    - committers
    - reviewers

- `/lgtm cancel` 或 GitHub Request Changes
  - reviewers
    - maintainers
    - techLeaders
    - coLeaders
    - committers
    - reviewers
  - **PR author**


## 设计思路

实现该插件不仅要考虑到它支持 `/lgtm` 这样的评论命令，而且要考虑它作为 code review 的协作工具怎么和 GitHub 本身的 review 功能结合起来。

以下情况下会触发该功能(**命令不区分大小写**)：

- 在 Comment 中使用 `/lgtm [cancel]`
- 使用 GitHub 本身 Approve/Request Changes 功能（如果打开了 review_acts_as_lgtm 选项）

**需要特别注意的是**：

- 该命令必须以 `/` 开始（**这是所有命令的基本规范**）

## 参数配置

| 参数名               | 类型     | 说明                                                              |
| -------------------- | -------- | ----------------------------------------------------------------- |
| repos                | []string | 配置生效仓库                                                      |
| review_acts_as_lgtm  | bool     | 是否将 GitHub Approve/Request Changes 视为有效的 `/lgtm [cancel]` |
| pull_owners_endpoint | string   | PR owners RESTFUL 接口地址                                        |

例如：

```yml
ti-community-lgtm:
  - repos:
      - ti-community-infra/test-live
      - ti-community-infra/tichi
      - ti-community-infra/ti-community-bot
      - ti-community-infra/ti-challenge-bot
      - tikv/pd
    review_acts_as_lgtm: true
    pull_owners_endpoint: https://prow.tidb.io/ti-community-owners # 你可以定义不同的获取 owners 的链接
```

## 参考文档

- [command help](https://prow.tidb.io/command-help?repo=ti-community-infra%2Ftest-live#lgtm)
- [代码实现](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/lgtm)

## Q&A

### 我是否可以 `/lgtm` 自己的 PR?

不可以，就算你拥有 reviewer 的权限也不会被记为一次有效的 code review。这就像在 GitHub 你无法 approve 自己的 PR 一样。

### 为什么 `/lgtm cancel` 会直接去掉我多次的 review 的结果？

因为当一个 reviewer 认为该代码存在问题并且需要重新 review 时，我们认为前面的 review 也是存在隐患的。

### 为什么我有了新的提交 lgtm 相关的标签还是保存？

这是因为目前 TiDB 社区的 code review 阶段较多，如果在有新的提交时立马取消该 lgtm 这会导致整个 PR review 过程周期很长， PR 合并困难。所以我们将这部分放宽松由 reviewer 和作者负责，在觉得需要重新 review 时可以自行 `/lgtm cancel`。

