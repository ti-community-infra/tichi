# size

## 设计背景

在 GitHub PR 列表里，我们无法直接得知 PR 代码改动的大小（增删行数）。但是我们有时候在安排 review 工作时可能会根据代码改动的大小来安排时间。

[size](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/size) 会根据 PR 代码增删行数为 PR 添加 size 标签。

## 设计思路

该插件由 Kubernetes 社区设计开发，实现十分简单。 size 通过检测 PR 增删行数给 PR 添加 `size/*` 标签，每个 size 标签都有对应的行数范围：

- `size/XS`: 0-9
- `size/S`: 10-29
- `size/M`: 30-89
- `size/L`: 89-269
- `size/XL`: 270-519
- `size/XXL`: 520+

## 参数配置

| 参数名 | 类型 | 说明            |
| ------ | ---- | --------------- |
| s      | int  | size/S 的行数   |
| m      | int  | size/M 的行数   |
| l      | int  | size/L 的行数   |
| xl     | int  | size/XL 的行数  |
| xxl    | int  | size/XXL 的行数 |

例如：

```yaml
size:
  s: 10
  m: 30
  l: 100
  xl: 500
  xxl: 1000
```

## 参考文档

- [size doc](https://prow.tidb.io/plugins?repo=ti-community-infra%2Ftichi)
- [代码实现](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/size)

## Q&A

### 它会跟着我 PR 的改动自动添加和移除 `size/*` 标签吗？

会，它会根据代码改动动态的添加和移除对应标签。