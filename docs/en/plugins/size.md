# size

## Design Background

In the GitHub PR list, we don't directly know the size of the PR code changes (the number of lines added or removed). However, we sometimes schedule our reviews based on the size of the changes.

[size](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/size) adds a size tag to the PR based on the number of lines added or deleted to the PR code.

## Design

This plugin was designed and developed by the Kubernetes community, and it's implementation is very simple. size adds `size/*` tags to a PR by detecting the number of lines added or deleted by the PR, and each size tag has a range of lines corresponding to.

- `size/XS`: 0-9
- `size/S`: 10-29
- `size/M`: 30-89
- `size/L`: 89-269
- `size/XL`: 270-519
- `size/XXL`: 520+

## Parameter Configuration

| Parameter Name | Type | Description                   |
| -------------- | ---- | ----------------------------- |
| s              | int  | size/S number of rows         |
| m              | int  | size/M number of rows         |
| l              | int  | number of rows for size/L     |
| xl             | int  | The number of rows in size/XL |
| xxl            | int  | size/XXL rows                 |

For example:

```yaml
size:
  s: 10
  m: 30
  l: 100
  xl: 500
  xxl: 1000
```

## Reference Documents

- [command help](https://prow.tidb.io/plugins?repo=ti-community-infra%2Ftichi)
- [code](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/size)

## Q&A

### Will it automatically add and remove `size/*` tags when I change my PR?

Yes, it will dynamically add and remove tags based on code changes.
