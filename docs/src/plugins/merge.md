# ti-community-merge

## 设计背景

在 TiDB 社区中，我们进行了多阶段的 code review 之后才能将代码合并。原来机器人会在 **committer** 使用 `/merge` 之后尝试重新运行所有测试之后合并。但是因为某些测试的不稳定，导致大量的合并失败重试，并且在重新合并时又需要将所有测试跑一遍。 `/merge` 作为一个一次性命令反复触发全部测试，导致整个合并周期变长。

ti-community-merge 采取不一样的策略，`/merge` 只负责打上 `status/can-merge` 的标签，然后机器人在所有 CI 通过时，会自动的将 PR 合并。**如果其中要求的一个不稳定测试没有通过，单独的运行重跑不稳定的测试即可**。

## 权限设计

该插件主要负责控制代码的合并，权限如下：

- `/merge` 
  - committers
    - maintainers
    - techLeaders
    - coLeaders
    - committers

- `/merge cancel` 
  - committers
    - maintainers
    - techLeaders
    - coLeaders
    - committers
  - **PR author**

## 设计思路

实现该插件需要考虑到它作为合并 PR 的最后关卡，我们需要严格控制 `status/can-merge` 标签的使用。尽量保证当我们打上标签之后（**请使用命令打标签，不要手动操作去添加该标签，这是 PR 合并过程中最敏感的一个标签**），确定所有的代码都是经过多人 review 有保障的。

考虑这样一个情况，当我们的 committer 在 code review 之后打上 `status/can-merge` 标签，测试也通过了，这个时候一般情况下就可以合并了。但是如果 PR 的作者在机器人合并之前（机器人每隔 1 分钟尝试扫描合并一次）提交了新的代码，**如果我们不自动去除 `status/can-merge` 标签，那这段新提交的代码就会在没有任何的 review 和保障之下测试通过之后合并**。

所以需要在有新的提交之后自动去除掉上一次通过 `/merge` 打上的标签。要求重新对该代码进行 code review。这样就保证了我们在 ti-community-lgtm 中不移除 LGTM 相关标签，但是也能在合并之前保证所有的代码都有 code review。

## 参数配置 

| 参数名               | 类型     | 说明                                                                                                                                    |
| -------------------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| repos                | []string | 配置生效仓库                                                                                                                            |
| store_tree_hash      | bool     | 是否将打上 `status/can-merge` 标签时的提交 hash 存储下来，当我们如果只是将 master 通过 GitHub 按钮合并进入当前 PR，那就可以保持住该标签 |
| pull_owners_endpoint | string   | PR owners RESTFUL 接口地址                                                                                                              |

例如：

```yml
ti-community-merge:
  - repos:
      - tidb-community-bots/test-live
      - tidb-community-bots/ti-community-prow
      - tidb-community-bots/ti-community-bot
      - tidb-community-bots/ti-challenge-bot
      - tikv/pd
    store_tree_hash: true
    pull_owners_endpoint: https://prow.tidb.io/ti-community-owners
  - repos:
      - tikv/community
      - pingcap/community
    store_tree_hash: true
    pull_owners_endpoint: https://bots.tidb.io/ti-community-bot
```

## 参考文档

- [command help](https://prow.tidb.io/command-help?repo=tidb-community-bots%2Ftest-live#merge)
- [代码实现](https://github.com/tidb-community-bots/ti-community-prow/tree/master/internal/pkg/externalplugins/merge)

## Q&A

### 我可不可以 `/merge` 自己的 PR?

不可以，因为 lgtm 控制宽松，当有新的提交时 lgtm 相关 label 不会自动取消，所以如果你是 committer 以上权限的人，你在有了新的提交之后，就可以直接重新打上 `status/can-merge` 标签，然后机器人就会自动合并该 PR。这样导致你最新的提交没有经过任何的 code review 就合并了。

### 我不使用 GitHub 的按钮，在本地去更新 master 到 PR，这样 `status/can-merge` 标签会消失吗？

会，因为当你在本地做完合并之后，我无法判断你是在合并 master 还是有新的提交。所以我们只信任使用 GitHub 更新按钮的合并提交。

### 我 rebase PR 会导致标签消失吗？

会，因为 rebase 之后所有提交的 hash 都会重新计算，我们存储在 comment 中的 hash 就会失效。


