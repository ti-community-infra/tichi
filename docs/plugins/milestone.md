# milestone

## 设计背景

通过命令让机器人来处理指派milestone的事务，同时可以添加一些说明来明确指派原因。

## 设计思路

通过在评论中`/milestone xxx`的命令来给Issue配置milestone。使用`/milestone clear`来清除配置。

***只有milestone维护人员才可以使用该命令。***

## 参数配置

无参数

## 参考文档

- [milestone doc](https://prow.tidb.io/plugins?repo=ti-community-infra%2Ftichi)
- [代码实现](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/milestone)

## Q&A

> 暂无
