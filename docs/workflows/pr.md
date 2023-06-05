# PR 工作流

## 设计背景

我们使用 prow 原生的 lgtm + approve 相关的插件来进行流程驱动，但我们在此基础上增加了可选配置多人 lgtm 的功能。

## PR 协作流程

大部分内容与上游 prow 原生的流程相同，建议先阅读[这里](https://github.com/kubernetes/community/blob/master/contributors/guide/owners.md#code-review-using-owners-files)。

以下是相对上游原生差异的部分：

- **Phase 1：** reviewers review 代码
  - 当 reviewer 进行 `lgtm` 动作时, 如果还没有达到配置的足够数量计数, bot 将会添加 `needs-*-more-lgtm` 标签。
  - 如果 reviewer `lgtm` 后整体的数量足够, bot 将会添加 `lgtm` 并移除 `needs-*-more-lgtm` 标签。
  - 任何 reviewer 进行 `/lgtm cancel` 则会重置 `lgtm` 的计数。


## 推荐配置项

### 推荐使用 Squash 模式合并代码

在合并方式上我们还是推荐采用 GitHub 的 Squash 模式进行合并，因为这是目前 TiDB 社区的传统，大家都会在 PR 中创建大量提交，然后在合并时通过 GitHub 自动进行 Squash。目前我们的 ti-community-merge 的设计也是为 Squash 模式服务，**如果不采用 Squash 模式，那么你在 PR 中就需要自己负责 rebase 或者 squash PR，这样会使我们存储提交 hash 的功能失效（详见 Q&A），最终导致 status/can-merge 因为有新的提交而自动取消**。所以我们强烈建议大家使用 Squash 模式进行协作。

### 如果仓库的 CI 任务是 prow 触发的需关闭 Require branches to be up to date before merging 分支保护选项

如果是 prow 触发的 CI 任务, 在 checkout 环节已经进行了与 base 的预合并再进行后续构建步骤。

## Q&A

### 为什么我自己 rebase 或者 squash 提交会导致 `lgtm` 被移除？

**因为我们目前存储的是打上 `lgtm` 标签时你 PR 中的最后一个提交的 hash**。当你 rebase PR 之后整个 hash 都会发生变化，所以会自动取消标签。当你自己 squash PR 的时候，因为我们存储的是最后一个提交的 hash 不是第一个提交的 hash，这样还是会导致自动取消标签。
