# size

## 设计背景

通常在PR列表里，我们无法直接得知PR提交的大小（增删行数）。size 插件可以帮助 reviewer 更方便地筛选PR。

## 设计思路

该插件由 Kubernetes 社区设计开发，实现十分简单。 size 通过检测PR行数给PR增加`size/*`标签，默认配置下每个 size 标签都有对应的行数范围：

- `size/XS`: 0-9
- `size/S`: 10-29
- `size/M`: 30-99
- `size/L`: 100-499
- `size/XL`: 500-999
- `size/XXL`: 1000+

## 参数配置

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

> 暂无
