# hold

## 设计背景

在我们做 Code Review 的过程中，可能会出现 PR 的改动没有问题，但是该 PR 会产生比较大的副作用需要其他人也参与进来评估是否可以合并并且何时合并的情况。

[hold](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/hold) 通过使用命令 `/hold [cancel]` 为 PR 添加或移除 `do-not-merge/hold` 标签,并且配合 [Tide](components/tide.md) 阻止 PR 的合并。

## 设计思路

该插件由 Kubernetes 社区设计开发，实现十分简单，就是通过 `/hold [cancel]` 添加或移除 `do-not-merge/hold` 标签来控制 PR 的合并。

## 参数配置

无配置

## 参考文档

- [command-help](https://prow.tidb.io/command-help#hold)
- [代码实现](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/hold)

## Q&A

### 我应该什么时候使用该功能？

代码没问题，你觉得可以同意这些改动，但是这些改动可能有一些副作用，需要更多人来仔细评估这些改动是否可以合并或者何时合并。