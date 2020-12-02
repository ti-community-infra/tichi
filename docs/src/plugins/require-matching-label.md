# require-matching-label

## 设计背景

在我们创建 Issue 或者提交 PR 时，有一些 label 是必须要添加的。比如针对 Issue 我们必须要打上 type/xxx 标签来区分该 Issue 属于什么类型。但是有些时候我们会忘了添加这些关键的标签，后续再去查找和添加这些 label 费时费力。所以我们希望能够自动化这个过程，督促大家尽快的添加相关 label。

[require-matching-label](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/require-matching-label) 通过正则的方式去匹配要求的 labels，**当 labels 缺失时打上对应缺失标签或者回复评论**。

## 设计思路

该插件由 Kubernetes 社区设计开发，他们在设计时，除了实现核心的根据正则去匹配 labels 之外，还考虑到了当 Issue 或者 PR 刚创建时可能会有自动化的工具去协助打上 labels，所以支持延时检测（默认为 5 秒）,在等待一段时间之后再去匹配和检测 labels 是否满足要求。

## 参数配置

| 参数名          | 类型   | 说明                                    |
| --------------- | ------ | --------------------------------------- |
| org             | string | 组织名                                  |
| repo            | string | 仓库名                                  |
| branch          | string | 分支                                    |
| prs             | bool   | 是否应用于 PR                           |
| issues          | bool   | 是否应用于 Issue                        |
| regexp          | string | label 正则表达式                        |
| missing_label   | string | 当找不到匹配的 label 时，打上该 label   |
| missing_comment | string | 当找不到匹配的 label 时，回复该 comment |
| grace_period    | string | 延迟检测时间（默认为 5 秒）             |

例如：
```yaml
- missing_label: needs-sig
  org: pingcap
  repo: tidb
  issues: true
  regexp: ^(sig|wg)/
  missing_comment: |
    There are no sig labels on this issue. Please add an appropriate label by using one of the following commands:
    - `/sig <group-name>`
    - `/wg <group-name>`
```

## 参考文档

- [require-matching-label doc](https://prow.tidb.io/plugins?repo=tidb-community-bots%2Fti-community-prow)
- [代码实现](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/require-matching-label)

## Q&A

### 是否可以在 Issue 或者 PR 创建一段时间之后再检查？

可以，配置 grace_period 参数即可。**默认为 5 秒**。