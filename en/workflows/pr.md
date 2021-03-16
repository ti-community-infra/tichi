# PR workflow

## Design Background

The collaboration process of PR has changed a bit after the introduction of Tide, but the basic framework has been kept. 
The main adjustment is in the PR merge phase, where instead of a one-time `/merge` command that triggers a bot to run tests and merge code, `/merge` is now only responsible for tagging `status/can-merge`. 
When the PR labels are satisfied and all tests pass, the PR will be merged automatically without human intervention.

**⚠️ Note: Please read the [Tide](en/components/tide.md), [ti-community-lgtm](en/plugins/lgtm.md), [ti-community-tars](en/plugins/tars.md), [ti-community-blunderbuss](en/plugins/blunderbuss.md), and [ti-community-merge](en/plugins/merge.md) chapters carefully before reading the following.**

## PR Collaboration Process

- Author Submit PR
- **Phase I：** Automatically request reviewers to PR ([ti-community-blunderbuss](en/plugins/blunderbuss.md) provide support)
  - Automatically request reviewers based on the sig to which the current PR belongs
  - Randomly select multiple reviewers based on ti-community-blunderbuss configuration
- **Phase II：** reviewers review code ([ti-community-lgtm](en/plugins/lgtm.md) provide support)
  - reviewer will look at the quality of the code, correctness, engineering considerations, etc.
  - If the reviewer finds no problems with the code, the reviewer will use `/lgtm` to agree to the changes; if the reviewer later finds that there are still problems with the code, they can cancel the changes by using `/lgtm cancel`
  - Once the reviewer uses the above command, the bot ti-chi-bot will automatically add or remove the lgtm-related labels 
- **Phase III：** committers review code ([ti-community-merge](en/plugins/merge.md) provide support)
  - committer to review the PR again, looking at dependencies with other features, forward/backward compatibility, etc.
  - If the committer thinks there will be no problems with the code changes, the committer will use `/merge` to agree to the merge; if the committer later thinks there are still problems with the code, it can cancel the merge by using `/merge cancel`
  - Once the committer uses the above command, the bot ti-chi-bot will automatically add or remove the `status/can-merge` label
- **Phase IV：** Auto-merge PR ([Tide](en/components/tide.md) provide support)
  - If all of the following requirements are met
    - All labels requested to already exist (e.g. status/can-merge)
    - No labels that prevent merging (e.g. do-not-merge/hold, needs-rebase)
  - If any of the following requirements are met
    - No CI testing for current PR
    - All tests triggered by the current PR have passed
  - After the above requirements are met, Tide automatically merges the PR

### Collaboration Flow Chart
- [Collaboration flowchart for PR requiring two lgtms(In Chinese)](https://viewer.diagrams.net/?highlight=0000ff&edit=_blank&layers=1&nav=1&title=pr-workflow.drawio#R7Vxdc5s4FP01emwGEB%2FiERw77W62022zm%2B3TDrEVmxaDF%2BM47q9fCYQBIWM5YOO6nslMLCGE4N5zdO%2BRAMDB%2FPUu9hazP6IJDoCmTF4BvAWapuqaSf7Rmk1WY2koq5jG%2FoQ1Kiq%2B%2BD8wq1RY7cqf4GWlYRJFQeIvqpXjKAzxOKnUeXEcravNnqOgetWFN8W1ii9jL6jXPvqTZMZqVdMuDrzH%2FnTGLo00Kzsw9%2FLG7E6WM28SrUtVcAjgII6iJPs1fx3ggD68%2FLk8ftg8Bvffzbvf%2Flz%2B5%2F3l%2Fv7w8e93WWejQ07Z3kKMw6TbrrWs6xcvWLHnxe412eQPMI5W4QTTThQA3VkyD8hPlfz8hpNkwwzurZKIVEVxMoumUegF91G0YO2eozBhzVRaxuHEoYYl5XHgLZf%2B%2BGHmh9mBkR%2Fk3ZMSOwuR0jKJo%2B9bC9LjW3PQYQXeEw5cb%2Fx9mg53EAVRTA6FUYhpVxPiEuyOiiEOi1pysSTe%2FEM7uzHy4lfWd1q4fa2UNqwkaRlmwWW0ise4oR1kAPHiKWb93ePhhxfzk%2Fq8mv47%2FaZ9%2BLieh8zSCr2vkpczu9%2FhaI7JIEmDGAde4r9UoeAxRE237banfop8ch%2BawtC%2FdX2GfdVSql1kA2VnFb5HfpSGUVSlHnmAd8K6dw4RcHXgqGBoATQAzpDWIAUgAwxNgEbAVsBQB64DHB1opjcnXuiGT8tFaizl0%2BcG%2F6ZetZ75Cf6y8FIzrQklVn2%2B7MvkqblT6sDMF%2Fa46GGu8oLjBL82Gjc%2FalaNhFhxXWK73I6zEtHpym53qBjyUKshgdVsaiPbSc03Asi6skyPLJNPzXtpRpekGeaJ75QbrcoX7EJvpaG8SfT8vMQVhtlLVdA8LVXZAqc3gU0YypHw%2Fv2ucwn4yL1frfh%2BAYUz837ZSZZ5oHKjGAidzOF1jvY1Q87hidG9TanZgjZY7o4BIAcszVIax8W3V%2B1Ke%2FIjG0Gn6MttWoJfjF98vCZ1KQ4RcEZgaNDowDFlpqNZNH9aLfcHBBX%2FpxgbeXM%2FoPf%2BHgcvOPHHniBs8AJ%2FGlLkEZfGsRhY5JJ%2BOCUlsyg9pHAmvHzEcMLi7A0F8YQiiCdQB%2FGEEIf1lORQuhQSooA4S3ZtN1VznPE29moipTJ5NSCiRUbQymR1OH5O4Uh8nQLSAs6A%2FtFAPY3eCSAJRF2TYpWE9K6RjjQDcGOguAedO5xf1h7SMfcWNKcIuoVPXK09qFOA5NVPShghpa%2BlIwVCaKFHgOxu1xc%2BzHq02IP5jm0KXdIURp%2Bm0HcF7q6TBgw2cBBTFGiNCVwLuC49hBzgKKnRvISECtro%2Fu6hDsOz5Cto9M1X6DqpcwCQAIrVJ1AM%2BTndodLcAXP6HlRU7X4qjOiiwFeEET7x6m5O72dSP1OQWJIg6TfyVQUrKnL6zz0VT6qmqKWJ1OtJRhk47MDcn0wyq%2BOl%2F8N7SvujxmBZPenccIFx2wQbtvDGTgZbIa1suAb3lFcDmZrfvTbyBhWhycN%2BORHhCFzKiQiG0reIcNVnj6bP5llgnveVzuotC5SdLHakHoWkqyHzWLRVX8PgUcNH39lt1yTdekeIC2UkF0PaasP5gGW1Yd0wOGC304bF05Qgw7uGVVy4JBNXwZZxldzCnaXI%2BWpn7lHXKtsIAHXfOksBwNT6FgDUX0MDU2W39Ki9qmBqfavNIJrP%2FSRh6b1Bc3i67cakPu%2FW5ZuzzOFNu3c3FwSel%2BjmshKWivp0c01E9heanduNSCHJOYuLzjcfV%2BuSo4CIrpl5m8zcEq1cijJzfutndwxZl12uqfkFp%2BbyU8WOhKOn5JzLqVF%2BHwcn5zaHQHik5BxyA1aMxnHx7S21241b4glZIJdfYniEJH2%2B9Ub%2BdlQs2rr9xlx47IXv5pjcbn3yPceEGMnu4OHR2l1savbh%2BYUsddjMcmzEbN8V24sYvU%2FEQAF%2FHRi8%2FHSZReaoTet%2B%2BhEn5q5mnhzHVxX4ILjB06jAqsIrMkeWgWG%2BUfrylQG4Y4NEoQxUTJGvpbW0sFo1cL4F8vhA10SWNel%2BJORSyzojYBtXFaGVimDLviRwNBVB62VD4dFpWXYL7ZFeANa5XBXx%2BzM6ylV13p%2FM5ly11r6aq%2B6dXuyT5LaircXNry8jYJtpNnVLX2WmidYtjSWLNjblIXKU0hXhMCUlMCulKyN9%2B1ltCj%2FrtMdj5ixWcaDOs4t1Y9T4xRbQi8FpMt3Ri91LtHiG71fIJ2a9ShlQ78VgZxreQ9lVaNh2FVoyvFdPvMsDij4l0aBsGZRb7dtSfJiGhYSg6SGbbmv%2FKWStIo%2FqTdeC%2FehaOXUesPJxrloYbLti0jYKFC8MdB0EqipPErA5CuRPOMmKBdy5XInsUlhm0cAL6aKP0lyEhtCsASo3qqZXZQR2xTMWBK%2BvpL2Fc%2FS2r9tIRgzbijezTucOs%2BOdAouuhBVMcKEEsGMfV7EIoOgVA%2BafcThjAhBtN6rLgm24%2FqoY1jOA3iTDPKwp2Zt%2BHk45KNY%2FxTfjTpAmCL%2Fy0tFGWVIsPpKZgbX41Cgc%2Fg8%3D)
  - If a new commit is found in the process, **it will keep the lgtm related label, but will automatically remove the `status/can-merge` label**
  - When some tests fail, **just trigger the failed test**

## Recommended configuration items

### Recommend using Squash mode to merge code

We recommend using GitHub's Squash mode for merging, because it's a tradition in the TiDB community to create a lot of commits in PR and then automatically Squash them through GitHub when merging. 
Our ti-community-merge is currently designed to work in Squash mode, **if you don't use Squash mode, then you are responsible for your own rebase or squash PR in PR, which will disable our ability to store commit hash (see Q&A for details) and eventually cause status/can-merge to be automatically removed due to a new commit**. 
So we strongly recommend that you use Squash mode for collaboration.

### It is recommended to turn on the `Require branches to be up to date before merging` branch protection option for small repositories

Turning this feature on solves the problem we mentioned in the Tide introduction:

> - PR1: Rename bifurcate() to bifurcateCrab()
> - PR2: Invoke bifurcate()
>   
> In this case, both PRs will be tested with the current master as the base branch, and both PRs will pass. However, once PR1 is merged into the master branch first, and the second PR is merged (because the test also passes), it causes a master error that `bifurcate` is not found.
> 

This feature requires the PR to merge the current master to the PR before merging, so that our PR uses the latest master branch as the Base test. However, turning this feature on has two effects:
1. PR cannot be merged automatically until the latest master is merged into PR
2. PR merge current master to PR causes `status/can-merge` label to disappear

**The first problem is solved by using [ti-community-tars](en/plugins/tars.md) to automatically update. The second problem is that we can identify the committer of the commit that updates master to PR using the GitHub update button as `web-flow`, so we can determine if we trust that commit based on the committer.**

## Q&A

### Why does my own rebase or squash commit cause `status/can-merge` to be removed?

**Because we are currently storing the hash of the last commit in your PR when tagged with `status/can-merge`**. 
When you rebase the PR, the entire hash will change, so it will be untagged automatically. 
When you squash the PR yourself, because we store the hash of the last commit and not the hash of the first commit, this will still result in automatic remove label.

