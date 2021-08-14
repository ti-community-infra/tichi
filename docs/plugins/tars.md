# ti-community-tars

## 设计背景

因为大多数的 CI 系统都是在PR 的当前 Base 上进行测试，所以当我们 PR 的 Base 落后时，就会出现以下问题：

- PR1: 重命名 bifurcate() 为 bifurcateCrab()
- PR2: 调用 bifurcate()
  
这个时候两个 PR 都会以当前 master 作为 Base 分支进行测试，两个 PR 都会通过。但是一旦 PR1 先合并入 master 分支，第二个 PR 合并之后（因为测试也通过了），就会导致 master 出现找不到 `bifurcate` 的错误。

为了解决该问题 GitHub 提供了一个名为 `Require branches to be up to date before merging` 的分支保护选项。打开该选项之后，PR 只有在使用最新 Base 分支的时候才能合并。**虽然这样能解决这个问题，但是需要人为手动的去点击 GitHub 按钮合并最新的 Base 分支到 PR，这是个机械且重复的事情**。

ti-community-tars 就是为了解决该问题而设计，它会在 PR 回复、更新或者 Base 分支有新提交时帮助我们自动合并最新的 Base 分支到 PR。除此之外，它也支持定期扫描所有配置了该插件的仓库的所有 PR，对它们 **逐个**进行更新。

## 设计思路

实现该插件需要考虑以下几种情形：
- PR 有回复或者更新
  - 当 PR 有回复或者更新时说明有人在关注该 PR，可能希望该 PR 尽快的合并，所以我们要尽快的响应并更新该 PR
- Base 分支有新的提交
  - 当 Base 分支有了新的提交之后，我们应该尽快查找其他可以合并的 PR 并将最新的 Base 更新到 PR 当中
  - 我们不能一次性将所有的 PR 都更新，因为我们每次最多只能合并一个 PR，所以我们应该选择创建时间最早并且可以合并的 PR 进行合并
- 定期的扫描并更新
  - 因为打开上面提到的选项就是为了保证 PR 合并最新的 Base 之后也通过测试，所以我们也要定期的合并最新的 Base 到这些 PR 尽快测试和解决可能的问题。这样定期的逐个更新这些 PR，也能防止合并队列因为前面的 PR 测试失败被阻塞

除此之外，大多数没有满足合并条件的 PR 并不希望进行自动更新。因为自动更新之后，我们在本地有新提交 push 的时候还需要拉取最新的更新。所以我们通过 label 配置项指定哪些 PR 需要被更新。

## 参数配置

| 参数名          | 类型     | 说明                                                                                                            |
| --------------- | -------- | --------------------------------------------------------------------------------------------------------------- |
| repos           | []string | 配置生效仓库                                                                                                    |
| message         | string   | 自动更新之后回复的消息                                                                                          |
| only_when_label | string   | 只有在 PR 添加该 label 的时候才帮忙更新，默认为 `status/can-merge`                                              |
| exclude_labels  | []string | 当 PR 有这些 labels 的时候不进行更新，默认为 `needs-rebase`/`do-not-merge/hold`/`do-not-merge/work-in-progress` |

例如：

```yaml
ti-community-tars:
  - repos:
      - ti-community-infra/test-dev
    only_when_label: "status/can-merge"
    exclude_labels:
      - needs-rebase
      - do-not-merge/hold
      - do-not-merge/work-in-progress
    message: "Your PR was out of date, I have automatically updated it for you."
```

## 参考文档

- [代码实现](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/tars)

## Q&A

### 多久会进行一次定期扫描？

目前为 20 分钟，每次更新每个仓库的不同分支的最早创建的那一个 PR。
