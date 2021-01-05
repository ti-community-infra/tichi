# rerere

[rerere](https://github.com/ti-community-infra/ti-community-prow/tree/master/internal/pkg/rerere) 是 ti-community-prow 的一个核心组件，rerere 会将代码 push 到专门用来重新测试的分支上进行重新测试。它将作为一个 [Prow Job](https://github.com/kubernetes/test-infra/blob/master/prow/jobs.md) 运行。

## 设计背景

开发该组件还是为了解决我们在 [Tide](components/tide.md) 中提到的多个 PR 合并的问题：

- PR1: 重命名 bifurcate() 为 bifurcateCrab()
- PR2: 调用 bifurcate()
  
这个时候两个 PR 都会以当前 master 作为 Base 分支进行测试，两个 PR 都会通过。但是一旦 PR1 先合并入 master 分支，第二个 PR 合并之后（因为测试也通过了），就会导致 master 出现找不到 `bifurcate` 的错误。

为了解决上述问题我们开发了 [tars](plugins/tars.md) 来自动的合并最新 的 Base 分支到 PR，这样能解决问题，但是对于大型仓库来说这不是一个高效的解决方案。**例如有 n 个同时可以 merge 的 PR，那就要跑 O(n^2) 次测试，大大浪费了 CI 资源。**

为了高效的合并 PR，并且节省测试资源，我们提出了[多个解决方案](https://github.com/ti-community-infra/configs/discussions/41)。最终决定利用 Prow Job 结合 Tide 来解决该问题。

## 设计思路

1. Prow Job 会利用 [clonerefs](https://github.com/kubernetes/test-infra/tree/master/prow/clonerefs) 工具将 PR 和最新的 Base 分支合并之后克隆到运行测试的 Pod，所以我们总是能够获取到已经合并了最新 Base 的代码库，rerere 可以将该代码 push 的重新测试分支进行合并之前的测试。
2. Porw Job 可以设置 max_concurrency 来控制该 CI 任务最多的执行个数，这是一个天然的 FIFO 队列，我们可以利用该功能对重新测试任务排队。
3. Tide 在代码合并之前，会检测所有的 Prow Job 都使用了最新的 Base 进行了测试。如果当前 CI Pod 使用的不是最新的 Base，Tide 会自动重新触发该测试，这样就确保了所有的 PR 在合并之前都会用最新的 Base 重新测试。
4. 在 rerere 中我们会将代码 push 到指定的测试分支，然后定期检查要求的 CI 是否都已经通过，当所有要求的 CI 都通过时，我们 rerere 的 Prow Job 就会通过测试。Tide 则会自动合并当前 PR。

## 参数配置

| 参数名           | 类型          | 说明                                               |
| ---------------- | ------------- | -------------------------------------------------- |
| retesting-branch | string        | 用于重新测试的分支名                               |
| retry            | int           | 当重新测试超时后，重试的次数                       |
| timeout          | time.Duration | 每次尝试重新测试的 timeout                         |
| labels           | []string      | 重新测试之前必须有的标签（例如：status/can-merge） |
| require-contexts | []string      | 要求必须通过的 CI 任务名                           |

例如：

```yaml
presubmits:
  tikv/tikv:
    - name: pull-tikv-master-rerere
      decorate: true
      trigger: "(?mi)^/(merge|rerere)\\s*$"
      rerun_command: "/rerere"
      max_concurrency: 1 # 最多同时运行一个任务
      branches:
        - ^master$
      spec:
        containers:
          - image: rustinliu/rerere-component:latest
            command:
              - rerere
            args:
              - --github-token-path=/etc/github/token
              - --dry-run=false
              - --github-endpoint=https://api.github.com
              - --retesting-branch=master-retesting
              - --timeout=40m
              - --require-contexts=tikv_prow_integration_common_test/master-retesting
              - --require-contexts=tikv_prow_integration_compatibility_test/master-retesting
              - --require-contexts=tikv_prow_integration_copr_test/master-retesting
              - --require-contexts=tikv_prow_integration_ddl_test/master-retesting
            volumeMounts:
              - name: github-token
                mountPath: /etc/github
                readOnly: true
        volumes:
          - name: github-token
            secret:
              secretName: github-token
```

## 参考文档
- [代码实现](https://github.com/ti-community-infra/ti-community-prow/tree/master/internal/pkg/rerere)

## Q&A

### 我每次提交都会触发 rerere 吗？

是的，但是在打上 `status/can-merge` 之前，我们不会真的 push 到测试分支进行测试，该测试会因为缺少要求的标签而跳过。等到使用 `/merge` 命令时，又会再次触发该 CI，此时因为已经有了 `status/can-merge` 我们才会真的进行重新测试。

### 如果测试失败了，还需要 `/merge` 吗？

不需要，使用 Tide 之后我们无需不停的使用 `/merge`，哪一个测试失败重新触发该测试即可。例如 rerere 测试失败，你只需要 `/rerere` 重新触发让它重新测试即可。

### 我人为的合并 PR 之后，会不会导致排队重新测试混乱出问题？

不会，除了你手动合并的 PR 会受到影响，其他 PR 会在合并之前检测到 Base 发生变化，Tide 会自动重新触发 rerere 进行测试。**强烈建议不要手动合并 PR，测试失败重新触发即可，当所有要求的 CI 通过时，Tide 会重新尝试合并。**