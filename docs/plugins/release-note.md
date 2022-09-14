# release-note

## 设计背景

在大型开源软件发布版本时一般都会提供一个发布说明来描述该版本的主要改动和更新。为了更好的收集这些信息，TiDB 社区会要求贡献者在 PR 中添加发布说明以备发布时整理和使用。

[release-note](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/releasenote) 会检测 PR 的 Body 中是否按照标准格式添加了发布说明，并且添加对应的标签来标记 PR。

## 设计思路

该插件由 Kubernetes 社区设计开发，主要负责检测在 PR 的 Body 中是否添加了如下格式的发布说明：

```go
noteMatcherRE = regexp.MustCompile(`(?s)(?:Release note\*\*:\s*(?:<!--[^<>]*-->\s*)?` + "```(?:release-note)?|```release-note)(.+?)```")
noneRe        = regexp.MustCompile(`(?i)^\W*NONE\W*$`)
```

我们推荐填写这种格式的发布说明：

```
    ```release-note
    Some release note.
    ```
```

使用 markdown 的代码块来组织发布说明，这样做的好处是格式简单并且在 markdown 渲染之后易读。**当我们按照这样的格式填写发布说明之后， release-note 插件就会为该 PR 添加 `release-note` 标签来标记该 PR 已经正确的添加了发布说明。**

除此之外，还考虑到一些代码的重构或者非代码相关的改动可能不需要填写发布说明，我们也可以用 None 来表示为无发布说明：

```
    ```release-note
    None
    ```
```

跟上面的格式一样，但是在代码块中填写 None 表示无需发布说明。**当我们将发布说明内容填写为 None 时，release-note 插件就会为该 PR 添加 `release-note-none` 标签来标记该 PR 无需添加发布说明。**

如果你觉得这种不需要发布说明的 PR 还要填写这个内容比较复杂，那你也可以直接使用 `/release-note-none` 命令来直接添加 `release-note-none` 标签。

最后，如果 release-note 插件检测不到任何的 release-note 代码块并且你也没有用命令标记该 PR 不需要发布说明，那么 release-note 插件就会为该 PR 添加 `do-not-merge/release-note-label-needed` 标签阻止 PR 的合并。

## 参数配置

无配置

## 参考文档

- [command help](https://prow.tidb.net/command-help#release_note_none)
- [代码实现](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/releasenote)