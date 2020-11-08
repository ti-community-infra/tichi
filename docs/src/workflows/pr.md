# PR workflow

## 设计背景

在引入 Tide 之后 PR 的协作流程发生了一些变化，但是基本的大框架还是保留了下来。主要的调整发生在 PR 合并阶段，从原来一次性的 `/merge` 命令去触发机器人运行测试合并代码变成了 `/merge` 只负责打上 `status/can-merge` 标签。当 PR 的标签满足要求并且所有测试通过后，PR 将会自动合并无需人为干预。

**⚠️ 注意：在阅读以下内容之前请先仔细阅读 [Tide](./../components/tide.md)、[ti-community-lgtm](./../plugins/lgtm.md) 和 [ti-community-merge](./../plugins/merge.md) 章节内容**。

## 推荐使用 Squash 模式合并代码

在合并方式上我们还是推荐采用 GitHub 的 Squash 模式进行合并，因为这是目前 TiDB 社区的传统，大家都会在 PR 中创建大量提交，然后再合并时通过 GitHub 自动进行 Squash。目前我们的 ti-community-merge 的设计也是为 Squash 模式服务，**如果不采用 Squash 模式，那么你在 PR 中就需要自己负责 rebase 或者 squash PR，这样会使我们存储提交 hash 的功能失效（详见 Q&A），最终导致你的标签被视为新的提交而自动取消**。所以我们强烈建议大家使用 Squash 模式进行协作。

## 推荐打开 Require branches to be up to date before merging 分支保护选项

打开这个功能就能解决我们在 Tide 介绍中提到的问题：

> - PR1: 重命名 bifurcate() 为 bifurcateCrab()
> - PR2: 调用 bifurcate()
>   
> 这个时候两个 PR 都会以当前 master 作为 Base 分支进行测试，两个 PR 都会通过。但是一旦 PR1 先合并入 master 分支，第二个 PR 合并之后（因为测试也通过了），就会导致 master 出现找不到 `bifurcate` 的错误。
> 

该功能要求 PR 在合并之前必须合并当前 master 到 PR，这样就保证我们的 PR 用了最新的 master 分支作为 Base 测试。但是打开这个功能之后有两个影响：
1. PR 在合并当前 master 到 PR 之前无法自动合并
2. PR 合并当前 master 到 PR 导致 `status/can-merge` 标签消失

**第一个问题我正在努力解决当中，会开发一个插件自动更新 master 到 PR。第二个问题已经解决，因为可以识别到使用 GitHub 更新按钮更新 master 到 PR 的提交的 committer 为 `web-flow`，所以可以根据 committer 来判断是否信任该提交。**

## 推荐协作流程

- [要求两个 lgtm 的 PR 的协作流程图](https://viewer.diagrams.net/?highlight=0000ff&edit=_blank&layers=1&nav=1&title=pr-workflow.drawio#R7Vxdc5s4FP01emwGEB%2FiERw77W62022zm%2B3TDrEVmxaDF%2BM47q9fCYQBIWM5YOO6nslMLCGE4N5zdO%2BRAMDB%2FPUu9hazP6IJDoCmTF4BvAWapuqaSf7Rmk1WY2koq5jG%2FoQ1Kiq%2B%2BD8wq1RY7cqf4GWlYRJFQeIvqpXjKAzxOKnUeXEcravNnqOgetWFN8W1ii9jL6jXPvqTZMZqVdMuDrzH%2FnTGLo00Kzsw9%2FLG7E6WM28SrUtVcAjgII6iJPs1fx3ggD68%2FLk8ftg8Bvffzbvf%2Flz%2B5%2F3l%2Fv7w8e93WWejQ07Z3kKMw6TbrrWs6xcvWLHnxe412eQPMI5W4QTTThQA3VkyD8hPlfz8hpNkwwzurZKIVEVxMoumUegF91G0YO2eozBhzVRaxuHEoYYl5XHgLZf%2B%2BGHmh9mBkR%2Fk3ZMSOwuR0jKJo%2B9bC9LjW3PQYQXeEw5cb%2Fx9mg53EAVRTA6FUYhpVxPiEuyOiiEOi1pysSTe%2FEM7uzHy4lfWd1q4fa2UNqwkaRlmwWW0ise4oR1kAPHiKWb93ePhhxfzk%2Fq8mv47%2FaZ9%2BLieh8zSCr2vkpczu9%2FhaI7JIEmDGAde4r9UoeAxRE237banfop8ch%2BawtC%2FdX2GfdVSql1kA2VnFb5HfpSGUVSlHnmAd8K6dw4RcHXgqGBoATQAzpDWIAUgAwxNgEbAVsBQB64DHB1opjcnXuiGT8tFaizl0%2BcG%2F6ZetZ75Cf6y8FIzrQklVn2%2B7MvkqblT6sDMF%2Fa46GGu8oLjBL82Gjc%2FalaNhFhxXWK73I6zEtHpym53qBjyUKshgdVsaiPbSc03Asi6skyPLJNPzXtpRpekGeaJ75QbrcoX7EJvpaG8SfT8vMQVhtlLVdA8LVXZAqc3gU0YypHw%2Fv2ucwn4yL1frfh%2BAYUz837ZSZZ5oHKjGAidzOF1jvY1Q87hidG9TanZgjZY7o4BIAcszVIax8W3V%2B1Ke%2FIjG0Gn6MttWoJfjF98vCZ1KQ4RcEZgaNDowDFlpqNZNH9aLfcHBBX%2FpxgbeXM%2FoPf%2BHgcvOPHHniBs8AJ%2FGlLkEZfGsRhY5JJ%2BOCUlsyg9pHAmvHzEcMLi7A0F8YQiiCdQB%2FGEEIf1lORQuhQSooA4S3ZtN1VznPE29moipTJ5NSCiRUbQymR1OH5O4Uh8nQLSAs6A%2FtFAPY3eCSAJRF2TYpWE9K6RjjQDcGOguAedO5xf1h7SMfcWNKcIuoVPXK09qFOA5NVPShghpa%2BlIwVCaKFHgOxu1xc%2BzHq02IP5jm0KXdIURp%2Bm0HcF7q6TBgw2cBBTFGiNCVwLuC49hBzgKKnRvISECtro%2Fu6hDsOz5Cto9M1X6DqpcwCQAIrVJ1AM%2BTndodLcAXP6HlRU7X4qjOiiwFeEET7x6m5O72dSP1OQWJIg6TfyVQUrKnL6zz0VT6qmqKWJ1OtJRhk47MDcn0wyq%2BOl%2F8N7SvujxmBZPenccIFx2wQbtvDGTgZbIa1suAb3lFcDmZrfvTbyBhWhycN%2BORHhCFzKiQiG0reIcNVnj6bP5llgnveVzuotC5SdLHakHoWkqyHzWLRVX8PgUcNH39lt1yTdekeIC2UkF0PaasP5gGW1Yd0wOGC304bF05Qgw7uGVVy4JBNXwZZxldzCnaXI%2BWpn7lHXKtsIAHXfOksBwNT6FgDUX0MDU2W39Ki9qmBqfavNIJrP%2FSRh6b1Bc3i67cakPu%2FW5ZuzzOFNu3c3FwSel%2BjmshKWivp0c01E9heanduNSCHJOYuLzjcfV%2BuSo4CIrpl5m8zcEq1cijJzfutndwxZl12uqfkFp%2BbyU8WOhKOn5JzLqVF%2BHwcn5zaHQHik5BxyA1aMxnHx7S21241b4glZIJdfYniEJH2%2B9Ub%2BdlQs2rr9xlx47IXv5pjcbn3yPceEGMnu4OHR2l1savbh%2BYUsddjMcmzEbN8V24sYvU%2FEQAF%2FHRi8%2FHSZReaoTet%2B%2BhEn5q5mnhzHVxX4ILjB06jAqsIrMkeWgWG%2BUfrylQG4Y4NEoQxUTJGvpbW0sFo1cL4F8vhA10SWNel%2BJORSyzojYBtXFaGVimDLviRwNBVB62VD4dFpWXYL7ZFeANa5XBXx%2BzM6ylV13p%2FM5ly11r6aq%2B6dXuyT5LaircXNry8jYJtpNnVLX2WmidYtjSWLNjblIXKU0hXhMCUlMCulKyN9%2B1ltCj%2FrtMdj5ixWcaDOs4t1Y9T4xRbQi8FpMt3Ri91LtHiG71fIJ2a9ShlQ78VgZxreQ9lVaNh2FVoyvFdPvMsDij4l0aBsGZRb7dtSfJiGhYSg6SGbbmv%2FKWStIo%2FqTdeC%2FehaOXUesPJxrloYbLti0jYKFC8MdB0EqipPErA5CuRPOMmKBdy5XInsUlhm0cAL6aKP0lyEhtCsASo3qqZXZQR2xTMWBK%2BvpL2Fc%2FS2r9tIRgzbijezTucOs%2BOdAouuhBVMcKEEsGMfV7EIoOgVA%2BafcThjAhBtN6rLgm24%2FqoY1jOA3iTDPKwp2Zt%2BHk45KNY%2FxTfjTpAmCL%2Fy0tFGWVIsPpKZgbX41Cgc%2Fg8%3D)
  - 流程中如果发现新的提交，**会保持 lgtm 相关 label，但是会自动取消 `status/can-merge` 标签**
  - 在某些测试不通过时，**只需触发失败的测试即可**

## Q&A

### 为什么我自己 rebase 或者 squash 提交会导致 `status/can-merge` 被移除？

**因为我们目前存储的是打上 `status/can-merge` 标签时你 PR 中的最后一个提交的 hash**。当你 rebase 之后整个 hash 都会发生变化，所以会自动取消标签。当你自己 squash 的时候，因为我们存储的是最后一个提交的 hash 不是第一个提交的 hash，这样还是会导致 hash 对不上自动取消标签。
