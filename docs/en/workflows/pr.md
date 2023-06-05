# PR Workflow 

## Design Background 

We use prow's native lgtm + approve related plug-ins to drive the process, but we have added the optional configuration of multi-person lgtm on this basis. 

## Most of the PR collaboration process 

It's the same as the upstream prow native process, it is recommended to read [here](https://github.com/kubernetes/community/blob/master/contributors/guide/owners.md#code-review -using-owners-files). 

The following is the part of the difference relative to the original upstream: 

- **Phase 1: ** reviewers review code 
  - when the reviewer performs the `lgtm` action, if the configured sufficient count has not been reached, the bot will add `needs-*-more -lgtm` label. 
  - If the overall number of reviewer `lgtm` is sufficient, the bot will add `lgtm` and remove `needs-*-more-lgtm` labels. 
  - Any reviewer doing `/lgtm cancel` will reset the count of `lgtm`. 


## Recommended configuration items 

### It is recommended to use Squash mode to merge code

In terms of merging, we still recommend using GitHub's Squash mode for merging, because this is the current tradition of the TiDB community. Everyone will create a large number of commits in the PR, and then automatically perform Squash through GitHub when merging. At present, our ti-community-merge is also designed to serve the Squash mode. **If you do not use the Squash mode, then you need to be responsible for rebase or squash PR in the PR, which will invalidate our function of storing and submitting the hash (details See Q&A), eventually causing status/can-merge to automatically cancel** because there are new commits. So we strongly recommend that you use Squash mode for collaboration. 

### If the CI task of the warehouse is triggered by prow, you need to close the Require branches to be up to date before merging branch protection option. 

If it is a CI task triggered by prow, the pre-merge with the base has been performed in the checkout link before subsequent construction steps . 

## Q&A 

### Why does my own rebase or squash commit cause `lgtm` to be removed? 

**Because we are currently storing the hash of the last commit in your PR when you tagged `lgtm`**. When you rebase the PR, the entire hash will change, so the tag will be automatically canceled. When you squash PR yourself, because we store the hash of the last submission instead of the hash of the first submission, this will still lead to automatic untagging.
