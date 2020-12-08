# ti-community-lgtm

## 设计背景

在 TiDB 社区中，我们采用了多阶段 code review 的方式来进行协作。一个 PR 一般会经过多个人的 review，才能达到合并的基本条件。例如：当 PR 被第一个人 review 之后，会为该 PR 打上 `status/LGT1` 的标签。然后当 PR 被第二个 review 之后，会为该 PR 打上 `status/LGT2` 的标签。每个 sig 都会设置默认需要的 LGTM 的个数，一般为 2 个。

ti-community-lgtm 是用来根据权限自动的为 PR 添加 LGTM 对应 label 的插件。它会作为一个独立的服务部署，由 Prow Hook 将 GitHub 的 webhook 事件转发给该插件进行处理。

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

实现该插件不仅要考虑到它作为 `/lgtm` 这样的评论命令，而且要考虑它作为 code review 的协作工具怎么和 GitHub 本身的 review 功能结合起来。**因为我们是在 GitHub 的基础上进行协作功能的拓展和增强，所以 GitHub 本身的设计逻辑和操作规范我们需要严格适配和遵循**。

在实现插件之前我们需要定义清楚如下三个事件：
- Issue Comment
![issue-comment.png](https://user-images.githubusercontent.com/29879298/100052235-75020b00-2e58-11eb-918b-4994d3263878.png)
- Single Review Comment
![single-review-comment.png](https://user-images.githubusercontent.com/29879298/100052023-0624b200-2e58-11eb-8b77-9ebd5754121d.png)
- GitHub Review 功能（包括：**Comment/Approve/Request changes 三个功能**）
![github-approve.png](https://user-images.githubusercontent.com/29879298/100052399-d3c78480-2e58-11eb-874d-0e7a7bed149b.png)

在考虑到 TiDB 社区原来在使用该功能的混乱状况之后，我们对 lgtm 事件的响应做了更加严格的限制，只有在以下情况下才会触发该功能(**命令不区分大小写**)：

- 在 Issue Comment 中使用 `/lgtm [cancel]`
- 在 Single Review Comment 中使用 `/lgtm [cancel]`
- 使用 GitHub 本身 Approve/Request Changes 功能(**⚠️注意：为了遵循 GitHub Review 功能的语义，我们忽略了其中的 Comment，因为 GitHub 对它的语义定义就是没有显式的 Approve**)

**需要特别注意的是**：

- 该命令必须以 `/` 开始（**这是所有命令的基本规范**）
- Review 功能中的 Comment 不会生效（使用 Review 功能请直接选择 Approve/Request Changes）

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
      - tidb-community-bots/test-live
      - tidb-community-bots/ti-community-prow
      - tidb-community-bots/ti-community-bot
      - tidb-community-bots/ti-challenge-bot
      - tikv/pd
    review_acts_as_lgtm: true
    pull_owners_endpoint: https://prow.tidb.io/ti-community-owners # 你可以定义不同的获取 owners 的链接
  - repos:
      - tikv/community
      - pingcap/community
    review_acts_as_lgtm: true
    pull_owners_endpoint: https://bots.tidb.io/ti-community-bot # 我们针对 community 做了 owners 的定制
```

## 参考文档

- [command help](https://prow.tidb.io/command-help?repo=tidb-community-bots%2Ftest-live#lgtm)
- [代码实现](https://github.com/tidb-community-bots/ti-community-prow/tree/master/internal/pkg/externalplugins/lgtm)

## Q&A

### 为什么我在使用 GitHub 的 Review 功能中的 Comment 时填写 `/lgtm` 无效？

因为我们要兼容 GitHub 对该功能本身的语义定义：`Submit general feedback without explicit approval.`，所以使用该功能不会打上 LGTM 相应标签。

### 为什么我使用 GitHub Review 功能的 Approve 并且在 comment 中填写 `/lgtm` 导致 Approve 无效？

因为过去一段时间在 TiDB 社区，lgtm 功能支持的触发条件太多，大家使用过程越来越随意和复杂，有时根本分不清楚该操作是否该记为一次有效的 lgtm。在这次的设计中我们希望保证该功能清晰明了并且兼容 GitHub 本身设计，在使用 Approve 时不再适配任何的 comment 中的 `/lgtm`，所以当你同时使用这两个功能时，机器人会直接忽略这次操作。**(该功能为暂时性限制，等到大家熟悉新的机器人，不会再使用 Review 功能中的 Comment 填写 `/lgtm` 之后移除)**.

### 我是否可以 `/lgtm` 自己的 PR?

不可以，就算你拥有 reviewer 的权限也不会被记为一次有效的 code review。这就像在 GitHub 你无法 approve 自己的 PR 一样。

### 为什么 `/lgtm cancel` 会直接去掉我多次的 review 的结果？

因为当一个 reviewer 认为该代码存在问题并且需要重新 review 时，我们认为前面的 review 也是存在隐患的。

### 为什么我有了新的提交 lgtm 相关的标签还是保存？

这是因为目前 TiDB 社区的 code review 阶段较多，如果在有新的提交时立马取消该 lgtm 这会导致整个 PR review 过程周期很长， PR 合并困难。所以我们将这部分放宽松由 reviewer 和作者负责，在觉得需要重新 review 时可以自行 `/lgtm cancel`。

