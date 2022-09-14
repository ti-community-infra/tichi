# ti-community-merge

## Design Background

In the TiDB community, we had to do a multi-stage code review before we could merge the code. Originally the bot would try to re-run all the tests after **committer** used `/merge` and then merge them. However, some tests were unstable, causing a lot of failed merge retries and the need to run all the tests again when re-merging. `/merge` repeatedly triggers all the tests as a one-time command, resulting in a long merge cycle.

ti-community-merge takes a different tack, `/merge` just labels `status/can-merge` and the bot automatically merges the PRs when all CIs pass. **If one of the requested unstable tests doesn't pass, just run a separate rerun of the unstable test**.

## Permission design

The plugin is mainly responsible for controlling the merging of codes with the following permissions:

- `/merge` 
  - committers
    - maintainers
    - techLeaders
    - coLeaders
    - committers

- `/merge cancel` 
  - committers
    - maintainers
    - techLeaders
    - coLeaders
    - committers
  - **PR author**

## Design

Considering that it is the final hurdle for merging PRs, we need to strictly control the use of the `status/can-merge` label. Try to make sure that when we label (**please use the command to label, do not add the label manually, it is one of the most sensitive labels in the PR merge process**) all the code is reviewed by multiple people and guaranteed.

Consider a situation where our committer is labeled with `status/can-merge` after the code review and the tests pass, so it is normally ready to merge. But if the author of the PR commits new code before the robot merges (the robot tries to scan for merges every 2 minutes), **if we don't automatically remove the `status/can-merge` label, the newly committed code will be merged without any review or guarantee after the tests pass**.

So we need to automatically remove the labels that were last labeled with `/merge` after a new commit is made. This ensures that we don't remove the LGTM-related labels in ti-community-lgtm, but also ensures that all code has code review before merging.

## Parameter Configuration 

| Parameter Name       | Type     | Description                                                                                                                                                                                  |
| -------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| repos                | []string | Repositories                                                                                                                                                                                 |
| store_tree_hash      | bool     | Whether or not to store the commit hash when you label `status/can-merge` so that we can keep that label when we just merge the latest Base branch into the current PR via the GitHub button |
| pull_owners_endpoint | string   | PR owners RESTFUL API URL                                                                                                                                                                    |

For example:

```yml
ti-community-merge:
  - repos:
      - ti-community-infra/test-live
      - ti-community-infra/tichi
      - ti-community-infra/ti-community-bot
      - ti-community-infra/ti-challenge-bot
      - tikv/pd
    store_tree_hash: true
    pull_owners_endpoint: https://prow.tidb.net/ti-community-owners
  - repos:
      - tikv/community
      - pingcap/community
    store_tree_hash: true
    pull_owners_endpoint: https://bots.tidb.io/ti-community-bot
```

## Reference Documents

- [command help](https://prow.tidb.net/command-help?repo=ti-community-infra%2Ftest-live#merge)
- [code](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/merge)

## Q&A

### Will the `status/can-merge` label disappear if I update the master to PR locally without using the GitHub button?

Yes, because when you do a merge locally, I can't tell if you're merging master or if there's a new commit. So we only trust merge commits that use the GitHub update button.

### Will my own manual rebase PR cause the labels to disappear?

Yes, because the hash of all commits will be recalculated after rebase, and the hash we stored in comment will be invalid.
