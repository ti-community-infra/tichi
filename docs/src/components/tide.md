# [Tide](https://github.com/kubernetes/test-infra/tree/master/prow/tide)

Tide 是 Prow 的一个组件，主要通过一些给定条件来管理 GitHub PR 池。它将自动重新检测符合条件的PR，并在它们通过检测时自动合并它们。

它具有一下特性：
- 自动运行批处理测试，并在可能的情况下将多个 PR 合并在一起。（如果你不使用 Prow 的 CI，这个功能是失效的）
- 确保在允许 PR 合并之前，针对最近的基本分支提交对 PR 进行测试。（如果你不使用 Prow 的 CI，这个功能是失效的）
- 维护一个 GitHub 状态上下文，该上下文指示每个 PR 是否在 PR 池中或缺少哪些要求。（这个就是指类似于其他 CI 在 PR 中汇报的状态，在该状态的 message 中会指定目前 PR 的状态）
- 支持使用特别的 GitHub Label 阻止 PR 合并到单个分支或整个存储库。
- Prometheus 指标。
- 支持具有“可选”状态，这些状态上下文对于合并不是强制的。
- 提供有关当前 PR 池的实时数据和尝试合并历史记录，可以在 [Deck](https://github.com/kubernetes/test-infra/tree/master/prow/cmd/deck)、[Tide Dashboard](https://prow.tidb.io/tide) 、[PR Status](https://prow.tidb.io/pr) 和 [Tide History](https://prow.tidb.io/tide-history) 中展示这些数据。
- 有效地进行扩展，使具有单个 bot 令牌的单个实例可以为满足合并条件的数十个组织和存储库提供合并自动化。每个不同的 'org/repo:branch' 组合都定义了一个不干扰的合并池，因此合并仅影响同一分支中的其他 PR。
- 提供可配置的合并模式（'merge', 'squash', or 'rebase'）。

## Tide 在 TiDB 社区

Tide 在 TiDB 社区使用基本正常，但是我们还是遇到了一个棘手的问题（**目前其他社区也还没解决该问题**）：

- PR1: 重命名 bifurcate() 为 bifurcateCrab()
- PR2: 调用 bifurcate()
  
这个时候两个 PR 都会以当前 master 作为 Base 分支进行测试，两个 PR 都会通过。但是一旦 PR1 先合并入 master 分支，第二个 PR 合并之后（因为测试也通过了），就会导致 master 出现找不到 `bifurcate` 的错误。

目前我们正在致力于解决这个问题，我会在推荐的工作流中介绍如何解决该问题。

**Kubernetes 社区目前没有这个问题，因为如果使用 Prow 的 CI 系统 Tide 会自动有最新的 master 作为 base 进行测试**。

## 我怎么样才能让我的 PR 合并？

如果你只是想让自己的 PR 合并，你看下面的文档就可以了。

### 从这几个地方获取 PR 状态

- PR 下面的 CI 状态上下文。状态要么告诉你的 PR 在合并池了，要么告诉你为什么它不在合并池中。 点击详情会跳转到 [Tide Dashboard](https://prow.tidb.io/tide)。例如：![example](https://user-images.githubusercontent.com/29879298/98230629-54037400-1f96-11eb-8a9c-1144905fbbd5.png)
- 在 [PR status](https://prow.tidb.io/pr) 中，你的每一个 PR 都会有一个卡片，每个卡片都显示了测试结果和合并要求。（推荐使用）
- 在 [Tide Dashboard](https://prow.tidb.io/tide) 中，显示每个合并池的状态,可以查看 Tide 当前正在做什么以及 PR 在重新检测队列中的位置。

### Q&A

#### 我的 PR 有没有在合并队列中？

如果 Tide 的状态是成功（绿色）那么它就已经在合并池了，如果是未成功（黄色）那么它不会进入合并池。

#### 我的 PR 为啥没有在合并队列中？

如果是你刚刚更新完 PR，请等待一下让 Tide 有时间去检测（默认一分钟）。

决定你的 PR 在不在队列中由以下两个部分决定：
- 检查你的 PR label 是否满足要求。
- 检查你要求的 CI 是否都通过了。

#### 为什么 PR 合并了，但是 Tide 的状态还是 Penning?

因为有些情况下可能它检测到了 PR 已经满足了要求，但是还没来得及将状态更新到 GitHub 就已经合并了。

## 其他参考资料
- [Maintainer's Guide to Tide](https://github.com/kubernetes/test-infra/blob/master/prow/cmd/tide/maintainers.md)
- [bors-ng](https://github.com/bors-ng/bors-ng)


