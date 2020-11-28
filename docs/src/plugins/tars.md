# ti-community-tars

## 设计背景

因为大多数的 CI 系统都是在当前 PR 上进行测试，所以当我们 PR 的 Base 落后时，就会出现以下问题：

- PR1: 重命名 bifurcate() 为 bifurcateCrab()
- PR2: 调用 bifurcate()
  
这个时候两个 PR 都会以当前 master 作为 Base 分支进行测试，两个 PR 都会通过。但是一旦 PR1 先合并入 master 分支，第二个 PR 合并之后（因为测试也通过了），就会导致 master 出现找不到 `bifurcate` 的错误。

为了解决该问题 GitHub 提供了一个名为 `Require branches to be up to date before merging` 的分支保护选项。打开该选项之后，PR 只有在使用最新 Base 分支的时候才能合并。**虽然这样能解决这个问题，但是需要人为手动的去点击 GitHub 按钮合并最新的 Base 分支到 PR，这是个机械且重复的事情**。

ti-community-tars 就是为了解决该问题而设计，它会在 PR 有回复或者更新时自动检测 PR 是否过期，然后帮助我们自动的合并最新的 Base 分支到 PR。除此之外，它也支持定期扫描所有配置了该插件的仓库的所有 PR，然后帮助我们更新已经过期的 PR。

## 设计思路

实现该插件，我们不仅需要在 PR 更新或者有回复时，去自动检测和更新。还要考虑到某些 PR 在 code review 结束之后不会再进行更新和回复，这些 PR 可能就会因为 `Require branches to be up to date before merging` 的选项要求无法合并。所以该插件还需要支持定期的扫描更新这些 PR。

除此之外，大多数没有满足合并条件的 PR 可能不希望进行自动更新。因为自动更新之后，我们在本地有新提交 push 的时候还需要拉取最新的更新。所以我们通过 label 配置项指定哪些 PR 需要被更新。

## 参数配置

| 参数名          | 类型     | 说明                                      |
| --------------- | -------- | ----------------------------------------- |
| repos           | []string | 配置生效仓库                              |
| message         | string   | 自动更新之后回复的消息                    |
| only_when_label | string   | 只有在 PR 被打上该 label 的时候才帮忙更新 |

例如：
```yaml
ti-community-tars:
  - repos:
      - tidb-community-bots/test-dev
    only_when_label: "status/can-merge"
    message: "Your PR has out-of-dated, I have automatically updated it for you."
```

## Q&A

### 多久会进行一次扫描？

目前为半小时，后期会根据使用仓库数量进行调整。
