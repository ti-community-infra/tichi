# ti-community-lgtm

## 设计背景

在 TiDB 社区中，我们采用了多阶段 code review 的方式来进行协作。一个 PR 一般会经过多个人的 review，才能达到合并的基本条件。例如：当 PR 被第一个人 review 之后，会为该 PR 打上 `status/LGT1` 的标签。然后当 PR 被第二个 review 之后，会为该 PR 打上 `status/LGT2` 的标签。每个 sig 都会设置默认需要的 LGTM 的个数，一般为 2 个。

ti-community-lgtm 是会根据命令和权限自动的为 PR 添加 LGTM 对应 label 的插件。它会作为一个独立的服务部署，由 Prow Hook 将 GitHub 的 webhook 事件转发给该插件进行处理。

## 权限设计

该插件主要负责 code review 的协作过程，权限如下：

- GitHub Approve
  - reviewers
    - maintainers
    - techLeaders
    - coLeaders
    - committers
    - reviewers

- GitHub Request Changes
  - reviewers
    - maintainers
    - techLeaders
    - coLeaders
    - committers
    - reviewers

## 设计思路

实现该插件主要考虑它作为 code review 的协作工具怎么和 GitHub 本身的 review 功能结合起来。

以下情况下会触发该功能：

- 使用 GitHub 的 Approve/Request Changes 功能

## 参数配置

| 参数名               | 类型     | 说明                                                              |
| -------------------- | -------- | ----------------------------------------------------------------- |
| repos                | []string | 配置生效仓库                                                      |
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
    pull_owners_endpoint: https://prow.tidb.io/ti-community-owners # 你可以定义不同的获取 owners 的链接
```

## 参考文档

- [command help](https://prow.tidb.io/command-help?repo=ti-community-infra%2Ftest-live#lgtm)
- [代码实现](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/lgtm)

## Q&A

### 我是否可以 Approve 自己的 PR?

不可以，在 GitHub 上你无法 approve 自己的 PR。

### 为什么 Request Changes 会直接去掉我多次的 review 的结果？

因为当一个 reviewer 认为该代码存在问题并且需要重新 review 时，我们认为前面的 review 也是存在隐患的。

### 为什么我有了新的提交 lgtm 相关的标签还是保存？

这是因为目前 TiDB 社区的 code review 阶段较多，如果在有新的提交时立马取消该 lgtm 这会导致整个 PR review 过程周期很长， PR 合并困难。所以我们将这部分放宽松由 reviewer 负责，通过 Request Changes 可以重置 review 状态。
