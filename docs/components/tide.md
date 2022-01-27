# [Tide](https://github.com/kubernetes/test-infra/tree/master/prow/tide)

Tide 是 Prow 的一个核心组件，主要通过一些给定条件来管理 GitHub PR 池。它将自动重新检测符合条件的PR，并在它们通过检测时自动合并它们。

它具有以下特性：

- 自动运行批处理测试，并在可能的情况下将多个 PR 合并在一起。（如果你不使用 Prow 的 CI，这个功能是失效的）
- 确保在允许 PR 合并之前，针对最近的基本分支提交对 PR 进行测试。（如果你不使用 Prow 的 CI，这个功能是失效的）
- 维护一个 GitHub 状态上下文，该上下文指示每个 PR 是否在 PR 池中或缺少哪些要求。（这个就是指类似于其他 CI 在 PR 中汇报的状态，在该状态的 message 中会指定目前 PR 的状态）
- 支持使用特别的 GitHub Label 阻止 PR 合并到单个分支或整个存储库。
- Prometheus 指标。
- 支持具有“可选”状态，这些状态上下文对于合并不是强制的。
- 提供有关当前 PR 池的实时数据和尝试合并历史记录，可以在 [Deck](https://github.com/kubernetes/test-infra/tree/master/prow/cmd/deck) 、[Tide Dashboard](https://prow.tidb.io/tide) 、[PR Status](https://prow.tidb.io/pr) 和 [Tide History](https://prow.tidb.io/tide-history) 中展示这些数据。
- 有效地进行扩展，使具有单个 bot 令牌的单个实例可以为满足合并条件的数十个组织和存储库提供合并自动化。每个不同的 `org/repo:branch` 组合都定义了一个互不干扰的合并池，因此合并仅影响同一分支中的其他 PR。
- 提供可配置的合并模式（`merge`, `squash`, or `rebase`）。

## Tide 在 TiDB 社区

### Tide 的定期扫描

正如其名 Tide (意为 "潮汐") ，Tide 不会在 PR 打开或有新的 commit 提交的时候立刻触发检查，它采取的策略是周期性地（每隔一两分钟）进行全量扫描，检查所有使用 Tide 托管的仓库当中，哪些 open PR 满足合并条件，如果满足条件的话将会送入合并池（队列）准备合并。

**因此，如果发现 PR 的 Checks 当中没有 tide 或者提示 "Waiting for status to be reported"，请先等一段时间。**

我们可以通过 `tide.sync_period` 配置来控制全量扫描的周期，配置示例如下：

```yaml
tide:
  sync_period: 2m
```

### 配置 PR 合并规则

我们可以通过对 `queries` 或 `context_options` 配置对 org、repo 或 branch 的 PR 的合并规则进行配置。

详细配置请参考 [Configuring Tide](https://github.com/kubernetes/test-infra/blob/master/prow/cmd/tide/config.md#configuring-tide) 文档。

配置示例：

```yaml
tide:
  queries:
  - repos:
      - pingcap/community
    includedBranches:
      - master      # 只有 master 分支上的 PR 会被合并。
    labels:
      - status/can-merge # 该仓库的 PR 只有在被打上 status/can-merge 的标签时才能合并。
    missingLabels:
      # 该仓库的 PR 有以下的标签时，PR 不会被和合并。
      - do-not-merge
      - do-not-merge/hold
      - do-not-merge/work-in-progress
      - needs-rebase
    reviewApprovedRequired: true    # PR 必须满足 GitHub 上的 Review 条件。

  context_options:
    orgs:
      pingcap:
        repos:
          community:
            # 要求该仓库所有分支的 PR 都必须通过 license/cla 的 CI。
            required-contexts:
              - "license/cla"
            branches:
              master:
                # 要求该仓库所有 master 的 PR 都必须通过 Sig Info File Format 的 CI。
                required-contexts:
                  - "Sig Info File Format"
```

### 查看 Tide 的工作状态

从这几个地方获取 PR 状态：

#### PR Status Context

Contributor 可以在 PR 下面的 CI 状态上下文当中查看 PR 的状态，例如：有哪些合并之前必须通过的 status check 还没有通过，PR 缺少哪些必要的标签，又或是 PR 上不能有哪些标签。如果各项条件都通过了，PR 会进入到合并池（队列）当中等待合并（提示：`In merge pool.`）。

![PR Status Context](https://user-images.githubusercontent.com/29879298/98230629-54037400-1f96-11eb-8a9c-1144905fbbd5.png)

点击详情会跳转到 [Tide Dashboard](https://prow.tidb.io/tide) 。

#### PR Status Page

Contributor 可以在 [PR status](https://prow.tidb.io/pr) 页面当中查看到自己的提交 PR 的状态，你的每一个 PR 都会有一个卡片，每个卡片都显示了测试结果和合并要求。（推荐使用）

#### Tide Dashboard

在 [Tide Dashboard](https://prow.tidb.io/tide) 中，显示每个合并池的状态,可以查看 Tide 当前正在做什么以及 PR 在重新检测队列中的位置。如果在一个合并池（队列）当中的一个 PR 长时间处于正在合并的状态，

### Tide 的合并方式

GitHub 为我们提供了 merge / [squash](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/about-pull-request-merges#squash-and-merge-your-pull-request-commits) / [rebase](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/about-pull-request-merges#rebase-and-merge-your-pull-request-commits) 三种 Pull Request 的合并方式。我们可以通过 `tide.merge_method.<org_or_repo_name>` 配置来设置 Tide 默认的 PR 合并方式，也可以通过使用添加 label 的方式来指定指定 PR 的合并方式。

相关的 label 通过 `squash_label`、`rebase_label`、`merge_label` 三个配置项来确定。

TiDB 社区使用 `tide/merge-method-squash`、`tide/merge-method-rebase`、`tide/merge-method-merge` 作为定义 Tide 合并 PR 方式的 label。

配置示例：

```yaml
tide:
  merge_method:
    pingcap/community: squash # 该仓库默认的合并方式为 squash。

  squash_label: tide/merge-method-squash
  rebase_label: tide/merge-method-rebase
  merge_label: tide/merge-method-merge
```

注意：在使用其它合并方式前，需要确保仓库的设置中允许使用该合并方式，**仓库的配置修改后不会立刻同步到 Tide，Tide 会每隔一小时去获取最新的仓库设置**。

![Merge Button Setting](https://user-images.githubusercontent.com/5086433/151337189-29ae600b-2c1d-4bb3-bf71-b852d556f3cf.png)


### 自定义 Commit Message Template 模板

在人工合并 PR 的时候，GitHub 为我们提供了一个页面表单来填写 PR 合并之后所产生的 final commit 的 message title 和 message body 部分。

![Commit Message Form](https://user-images.githubusercontent.com/5086433/151338288-63da93b5-cd35-4622-842f-7f16fcc299f7.png)

现在我们使用 Tide 来帮助我们进行 PR 自动合并，Tide 为我们提供了 [`commit_message_template`](https://github.com/ti-community-infra/configs/blob/main/prow/config/config.yaml#:~:text=merge_commit_template) 配置来自定义 commit message title 和 commit message body 的模板。

```yaml
tide:
  merge_commit_template:
    ti-community-infra/test-dev:
      title: "{{ .Title }} (#{{ .Number }})"
      body: |
        {{ .Body }}
```

我们可以为整个 org 的 repos 或是单个 repo 来定义 commit message template，例如：上面配置，`title` 与 `body` 配置当中的内容会经过 go [text template](https://pkg.go.dev/text/template) 的处理，填充 [Pull Request](https://pkg.go.dev/k8s.io/test-infra/prow/tide#PullRequest) 相关的数据，生成最终的 commit message。

如果 `title` 或 `body` 配置项为空，合并 PR 的时候 GitHub 会使用 默认的 commit message (例如：[merge-message-for-a-squash-merge](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/about-pull-request-merges#merge-message-for-a-squash-merge) ) 作为 commit message 的 title 和 body，如果希望填写空的 commit message body，可以将 `body` 配置设置为空字符串 `" "`。

#### 工具函数

为了扩展 commit message 模板的能力，TiDB 社区在 [自定义的 Tide 组件](https://github.com/ti-community-infra/test-infra/tree/ti-community-custom/prow/tide) 当中为 text template 注入了一些实用的工具函数：

##### `.ExtractContent <regexp> <source_text>`

`.ExtractContent` 函数可以通过正则表达式来提取文本内容，当正则表达式中包含一个名为 `content` 的命名分组时, 该函数只返回 [命名分组](https://pkg.go.dev/regexp#Regexp.SubexpNames) 所匹配的内容，如果没有则返回整个正则表达式所匹配的内容。

配置示例：

```yaml
tide:
  merge_commit_template:
    ti-community-infra/test-dev:
      title: "{{ .Title }} (#{{ .Number }})"
      body: |
        {{- $body := print .Body -}}
        {{- $description := .ExtractContent "(?i)\x60\x60\x60commit-message(?P<content>[\\w|\\W]+)\x60\x60\x60" $body -}}
        {{- if $description -}}{{- "\n\n" -}}{{- end -}}
        {{- $description -}}
```

##### `.NormalizeIssueNumbers <source_text>`

`.NormalizeIssueNumbers` 函数用于从文本内容单中提取 issue number，并对其进行格式化处理，返回一个 issue number 的对象数组。

Linked issue number 必须使用 GitHub 所支持的关键字或约定的 `ref` 关键字作为前缀。

配置示例：

```yaml
tide:
  merge_commit_template:
    ti-community-infra/test-dev:
      title: "{{ .Title }} (#{{ .Number }})"
      body: |
        {{- $body := print .Body -}}
        {{- $issueNumberLine := .ExtractContent "(?im)^Issue Number:.+" $body -}}
        {{- $numbers := .NormalizeIssueNumbers $issueNumberLine -}}
        {{- if $numbers -}}
          {{- range $index, $number := $numbers -}}
            {{- if $index }}, {{ end -}}
            {{- .AssociatePrefix }} {{ .Org -}}/{{- .Repo -}}#{{- .Number -}}
          {{- end -}}
        {{- else -}}
          {{- " " -}}
        {{- end -}}
```

在上述配置当中，如果 PR body 当中存在一行 `Issue Number: close #123, ref #456`, 经过模板处理将会返回 `close org/repo#123, ref org/repo#456` 的文本结果。

##### `.NormalizeSignedOffBy`

该函数用于对 PR commits 当中的签名 (`Signed-off-by: `) 信息进行格式化处理，并返回一个 signed-author 对象数组。

在一些开源仓库当中会使用 [DCO 协议](https://wiki.linuxfoundation.org/dco) 替代 [CLA 协议](https://en.wikipedia.org/wiki/Contributor_License_Agreement) , 前者会要求 Contributor 在 Commit 当中填写 `Signned-off-by: ` 行来表明其接受该协议内容。

在使用 squash 方式合并 PR 时，PR 当中的多个 commit 会先合并为一个 squashed commit 再合入 base 分支。为了简化 squashed commit 的 message，我们可以通过 `.NormalizeSignedOffBy` 函数对多个 commit 的 `Signed-off-by: `信息进行去重与合并。

配置示例：

```yaml
tide:
  merge_commit_template:
    ti-community-infra/test-dev:
      title: "{{ .Title }} (#{{ .Number }})"
      body: |
        {{- $signedAuthors := .NormalizeSignedOffBy -}}
        {{- if $signedAuthors -}}{{- "\n\n" -}}{{- end -}}
        {{- range $index, $author := $signedAuthors -}}
        {{- if $index -}}{{- "\n" -}}{{- end -}}
        {{- "Signed-off-by:" }} {{ .Name }} <{{- .Email -}}>
        {{- end -}}
```

##### `.NormalizeCoAuthorBy`

该函数用于对 PR 共同作者的信息进行格式化处理，返回一个 co-author 对象数组。

一个 PR 当中可能会存在来自不同作者的多个 commit，同一个 commit 也可能存在多个作者（GitHub 约定可以在 commit message 当中使用 [`Co-authored-by：`](https://docs.github.com/en/pull-requests/committing-changes-to-your-project/creating-and-editing-commits/creating-a-commit-with-multiple-authors) 行来声明 commit 的共同作者）。

当我们使用 squash 方式合并 PR 时，PR 当中的多个 commit 会先合并为一个新的 squashed commit 再合入 base 分支，为了简化 squashed commit 的 message，我们可以通过 `.NormalizeCoAuthorBy` 函数对多个 commit 的工作作者信息进去去重与合并。

**该函数会认为 PR 当中除 PR 作者以外的 commit author 或 co-author 都是 squashed commit 的 co-author。**

配置示例：

```yaml
tide:
  merge_commit_template:
    ti-community-infra/test-dev:
      title: "{{ .Title }} (#{{ .Number }})"
      body: |
        {{- $coAuthors := .NormalizeCoAuthorBy -}}
        {{- if $coAuthors -}}{{- "\n\n" -}}{{- end -}}
        {{- range $index, $author := $coAuthors -}}
          {{- if $index -}}{{- "\n" -}}{{- end -}}
          {{- "Co-authored-by:" }} {{ .Name }} <{{- .Email -}}>
        {{- end -}}
```

### PR 合并与 CI 测试

Tide 在 TiDB 社区使用基本正常，但是还是遇到了一个棘手的问题（**目前其他社区也还没解决该问题**）：

- PR1: 重命名 bifurcate() 为 bifurcateCrab()
- PR2: 调用 bifurcate()

这个时候两个 PR 都会以当前 master 作为 Base 分支进行测试，两个 PR 都会通过。但是一旦 PR1 先合并入 master 分支，第二个 PR 合并之后（因为测试也通过了），就会导致 master 出现找不到 `bifurcate` 的错误。

我会在推荐的 PR 工作流中介绍如何解决该问题。

**Kubernetes 社区目前没有这个问题，因为如果使用 Prow 的 CI 系统 Tide 会自动用最新的 base 分支进行测试**。


### Q&A

#### 我的 PR 有没有在合并队列中？

如果 Tide 的状态是成功（绿色）那么它就已经在合并池了，如果是未成功（黄色）那么它不会进入合并池。

#### 我的 PR 为啥没有在合并队列中？

如果是你刚刚更新完 PR，请等待一下让 Tide 有时间去检测（默认一分钟）。

你的 PR 在不在队列中由以下两个要求决定：
- 检查你的 PR label 是否满足要求。
- 检查要求的 CI 是否都通过了。

#### 为什么 PR 合并了，但是 Tide 的状态还是 Penning?

因为有些情况下可能它检测到了 PR 已经满足了要求并且进行了合并，但是还没来得及将状态更新到 GitHub 就已经合并了。

#### 如果我的 PR 的其中某个测试失败了，我是需要全部重新跑，还是只跑一个？

只需要重新跑失败的测试即可。

## 参考资料

- [Maintainer's Guide to Tide](https://github.com/kubernetes/test-infra/blob/master/prow/cmd/tide/maintainers.md)
- [bors-ng](https://github.com/bors-ng/bors-ng)


