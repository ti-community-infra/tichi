# PR 工作流

## 设计背景

我们使用 prow 原生的 lgtm + approve 相关的插件来进行流程驱动，但我们在此基础上增加了可选配置多人 lgtm 的功能。

## PR 协作流程

- **Phase 1：** reviewers review 代码
  - 当 reviewer 进行 `lgtm` 动作时, 如果还没有达到配置的足够数量计数, bot 将会添加 `needs-*-more-lgtm` 标签。
  - 如果 reviewer `lgtm` 后整体的数量足够, bot 将会添加 `lgtm` 并移除 `needs-*-more-lgtm` 标签。
  - 任何 reviewer 进行 `/lgtm cancel` 则会重置 `lgtm` 的计数。

- 作者提交 PR 。
- 阶段 0：自动化建议 PR 的 [reviewers][reviewer-role] 和 [approvers][approver-role]
  - 确定最接近被更改代码的 OWNERS 文件集。
  - 至少选择两个建议的审阅者，尝试为每个叶子 OWNERS 文件找到一个独特的审阅者，并请求他们对 PR 进行审阅。
  - 从每个 OWNERS 文件中选择建议的批准人，并将他们列在 PR 的评论中。
- 阶段 1：人工审查 PR
  - 审阅者寻找一般的代码质量、正确性、健全的软件工程、风格等。
  - 组织中的任何人都可以担任审阅者，但打开 PR 作者除外。
  - 如果代码更改对他们来说很好，审阅者会在 PR 中 或者在 review 中评论 `/lgtm`；如果他们改变主意，他们可以评论 `/lgtm cancel`；- 如果未达到配置的足够 lgtm 计数，机器人将添加 `needs-*-more-lgtm` 标签。
  - 如果有足够多的审阅者已经 `/lgtm` [prow](https://prow.tidb.net) ([@ti-chi-bot](https://github.com/apps/ti-chi-bot)) 会为 PR 添加 `lgtm` 标签并删除 `needs-*-more-lgtm` 标签;
  - 任何有效的 reviewer 或者 PR 作者进行 `/lgtm cancel` 都会重置 lgtm 计数。
- 阶段 2：人工批准 PR
  - PR作者可向 PR 评论 `/assign`,这将推荐批准者，也可选择通知他们（例如：“pinging @foo for approval”）。
  - 只有相关 OWNERS 文件中列出的人，无论是直接还是通过别名，如上所述，都可以充当批准人，包括打开 PR 的个人。
  - 审批者寻找整体验收标准，包括与其他功能的依赖性、向前/向后兼容性、API 和标志定义等。
  - 如果代码更改对他们来说很好，批准者会/approve输入 PR 评论或评论；如果他们改变主意，他们可以 `/approve cancel`
  - [prow](https://prow.tidb.net) ([@ti-chi-bot](https://github.com/apps/ti-chi-bot)) 更新了它在 PR 中的评论以表明哪些批准者仍然需要批准。
  一 旦所有批准者（来自每个先前确定的 OWNERS 文件的批准者）都已批准，[prow](https://prow.tidb.net) ([@ti-chi-bot](https://github.com/apps/ti-chi-bot)) 应用 `approved` 标签。
- 阶段 3：自动化合并 PR：
  - 如果以下所有情况都为真：
    - 所有必需的标签都存在（例如：lgtm，approved）
    - 缺少任何阻塞标签（例如：没有do-not-merge/hold, needs-rebase）
  - 如果以下任何一项为真：
    - 没有为此 repo 配置的预提交 prow 作业
    - 有为此 repo 配置的预提交 prow 作业，它们在最后一次自动重新运行后全部通过
  - 然后 PR 会自动合并

> 基于 [kubernetes 社区 PR 评审流程](https://github.com/kubernetes/community/blob/master/contributors/guide/owners.md#code-review-using-owners-files)修改调整而来。


## 推荐配置项

### 推荐使用 Squash 模式合并代码

在合并方式上我们还是推荐采用 GitHub 的 Squash 模式进行合并，因为这是目前 TiDB 社区的传统，大家都会在 PR 中创建大量提交，然后在合并时通过 GitHub 自动进行 Squash。目前我们的 ti-community-merge 的设计也是为 Squash 模式服务，**如果不采用 Squash 模式，那么你在 PR 中就需要自己负责 rebase 或者 squash PR，这样会使我们存储提交 hash 的功能失效（详见 Q&A），最终导致 status/can-merge 因为有新的提交而自动取消**。所以我们强烈建议大家使用 Squash 模式进行协作。

### 如果仓库的 CI 任务是 prow 触发的需关闭 Require branches to be up to date before merging 分支保护选项

如果是 prow 触发的 CI 任务, 在 checkout 环节已经进行了与 base 的预合并再进行后续构建步骤。

## Q&A

### 为什么我自己 rebase 或者 squash 提交会导致 `lgtm` 被移除？

**因为我们目前存储的是打上 `lgtm` 标签时你 PR 中的最后一个提交的 hash**。当你 rebase PR 之后整个 hash 都会发生变化，所以会自动取消标签。当你自己 squash PR 的时候，因为我们存储的是最后一个提交的 hash 不是第一个提交的 hash，这样还是会导致自动取消标签。
