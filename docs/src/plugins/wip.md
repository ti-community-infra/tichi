# wip(Work In Process)

## 设计背景

在我们提交 PR 时，可能对于一些比较复杂问题的修复，我们需要多次提交修改才能完成 PR。在我们还在修改 PR 的过程中，我们希望 reviewers 不要来做 Code Review。

[wip](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/wip) 通过检测 PR 是否处于 draft 状态或 PR 标题上是否包含 `WIP` 来添加或移除 `do-not-merge/work-in-progress` 标签来配合 [Tide](../components/tide.md) 阻止 PR 的合并。

## 设计思路

该插件由 Kubernetes 社区设计开发，实现十分简单，当我们的 PR 处于 draft 状态或 PR 标题上包含 `WIP` 时自动添加 `do-not-merge/work-in-progress` 标签。

## 参数配置

无特殊配置

## 参考文档

- [wip doc](https://prow.tidb.io/plugins?repo=tidb-community-bots%2Fti-community-prow)
- [代码实现](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/wip)

## Q&A

### 它会跟着我 PR 的状态或标题的变化自动添加和移除 `do-not-merge/wip` 标签吗？

会，当你的 PR 不是 draft 状态或者标题不包含 `WIP` 时，它会自动移除该标签。