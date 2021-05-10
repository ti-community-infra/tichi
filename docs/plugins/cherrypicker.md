# ti-community-cherrypicker

## 设计背景

在 TiDB 社区中，几个大型仓库都有多个分支在维护。当我们对 master 的代码进行修改并创建 PR 之后，这些改动可能也需要应用到其他的分支。依靠人工手动的去 cherry-pick 会产生巨大的工作量并且容易出错。

ti-community-cherrypicker 将帮助我们自动的 cherry-pick PR 的改动到另外一个分支并自动创建 PR。**另外它还支持代当码冲突时采用 3-way 合并的方式将代码强制 cherry-pick 到 Base 分支。**

## 权限设计

该插件主要负责 cherry-pick 代码并创建 PR，权限如下：

- `allow_all` 配置为 true，所有 GitHub 用户都可以触发 `/cherry-pick some-branch`
- `allow_all` 配置为 false，则只有该 repo 所在 Org 的成员可以触发 `/cherry-pick some-branch`

## 设计思路

实现该插件主要考虑 PR 的以下两种情况：

- PR 无冲突
  - 我们可以直接下载 [GitHub 提供的 patch 文件](https://stackoverflow.com/questions/6188591/download-github-pull-request-as-unified-diff) 进行 3-way 模式的 [git am](https://git-scm.com/docs/git-am) 操作。
- PR 有冲突
  - 我们无法直接应用 patch，因为 patch 中的 commits 会被逐个应用，这个过程中可能会多次冲突，解决冲突的过程会十分复杂。
  - 我们可以直接 cherry-pick 当前 PR 合并时在 GitHub 上产生的 merge_commit_sha(rebase/merge/squash 合并方式都会产生该提交)，这样我们就可以一次性将整个 PR cherry-pick 到 Base 分支，同时只需要解决一次冲突。

注意：**以上的`解决冲突`是指该工具将冲突代码直接 `git add` 然后提交到新的 PR 中，而不是真的修改代码解决冲突问题**。

除了实现 cherry-pick 的核心功能之外，它还支持了一些其他功能：

- 使用 labels 来标记需要 cherry-pick 到哪些分支
- 复制当前 PR 的 reviewers 到 cherry-pick 的 PR 
- 将 cherry-pick 的 PR 分配给作者或者请求人（请求 cherry-pick 的人）
- 复制当前 PR 已有的 labels

## 参数配置 

| 参数名                   | 类型     | 说明                                                                            |
| ------------------------ | -------- | ------------------------------------------------------------------------------- |
| repos                    | []string | 配置生效仓库                                                                    |
| allow_all                | bool     | 是否允许非 Org 成员触发 cherry-pick                                             |
| create_issue_on_conflict | bool     | 当代码冲突时，是否创建 Issue 来跟踪，如果为 false 则会默认提交冲突代码到新的 PR |
| label_prefix             | string   | 触发 cherry-pick 的 label 的前缀，默认为 `cherrypick/`                          |
| picked_label_prefix      | string   | cherry-pick 创建的 PR 的 label 前缀（例如：`type/cherry-pick-for-release-5.0`） |
| exclude_labels           | []string | 一些不希望被该插件自动复制的 labels （例如：一些控制代码合并的 labels）         |

例如：

```yml
ti-community-cherrypicker:
  - repos:
      - pingcap/dumpling
    label_prefix: needs-cherry-pick-
    allow_all: true
    create_issue_on_conflict: false
    excludeLabels:
      - status/can-merge
      - status/LGT1
      - status/LGT2
      - status/LGT3
```

## 参考文档

- [command help](https://prow.tidb.io/command-help#cherrypick)
- [代码实现](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/cherrypicker)

## Q&A

### 如果机器人 cherry-pick 的 PR 和目标分支代码有冲突，我怎么样才能修改该 PR？

**如果你对该仓库有写权限，则可以直接修改该 PR。** 

GitHub 支持[维护者直接修改 fork 仓库的代码](https://docs.github.com/en/github/collaborating-with-issues-and-pull-requests/allowing-changes-to-a-pull-request-branch-created-from-a-fork)。机器人在创建 PR 时会默认打开该选项，以供维护者对 PR 进行修改。

### 我如何 checkout 机器人的 PR 进行修改？

GitHub 已经在 PR 页面中推荐你使用 GitHub 官方的 [cli](https://github.com/cli/cli) 进行 checkout 和修改。**详情请见 PR 页面右上角的 `Open with` 下拉框。**

### 为什么机器人会丢掉我的一些 commits？

这种情况只会在 merge Base 的 commit 中进行代码修改时发生，原因是 GitHub 在生成 patch 时会自动删除 merge Base 的提交。

例如：
> 
> 在该 [PR](https://github.com/pingcap/dm/pull/1638) 中有 8 个 commits，但是它的 [patch](https://patch-diff.githubusercontent.com/raw/pingcap/dm/pull/1638.patch) 中只有 5 个提交。**因为 GitHub 自动删除了 merge master 的提交。**
> 
> 可以看到 cherry-pick 的 [PR](https://github.com/pingcap/dm/pull/1650) 也只有 5 个提交，而且因为原 PR 中的这个[提交](https://github.com/pingcap/dm/pull/1638/commits/8c08720653a6904a029e76bd66d499ef73c385fc)不仅 merge 了 master 而且还对代码进行了修改，最终导致该提交被丢失。

**所以建议大家不要在 merge Base 的提交中修改代码，这样会导致代码丢失。**
