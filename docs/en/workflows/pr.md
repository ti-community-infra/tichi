# PR Workflow 

## Design Background 

We use prow's native lgtm + approve related plug-ins to drive the process, but we have added the optional configuration of multi-person lgtm on this basis. 

## Main PR Collaboration Process
- The **author** submits a PR
- Phase 0: Automation suggests **[reviewers][reviewer-role]** and **[approvers][approver-role]** for the PR
  - Determine the set of OWNERS files nearest to the code being changed
  - Choose at least two suggested **reviewers**, trying to find a unique reviewer for every leaf
    OWNERS file, and request their reviews on the PR
  - Choose suggested **approvers**, one from each OWNERS file, and list them in a comment on the PR
- Phase 1: Humans review the PR
  - **Reviewers** look for general code quality, correctness, sane software engineering, style, etc.
  - Anyone in the organization can act as a **reviewer** with the exception of the individual who
    opened the PR
  - If the code changes look good to them, a **reviewer** types `/lgtm` in a PR comment or review;
    if they change their mind, they `/lgtm cancel`;
    if the configured sufficient lgtm count has not been reached, the bot will add `needs-*-more -lgtm` label.    
  - If sufficient **reviewers** have `/lgtm`'ed, [prow](https://prow.tidb.net)
    ([@ti-chi-bot](https://github.com/apps/ti-chi-bot)) applies an `lgtm` label and remove `needs-*-more-lgtm` label to the PR;
  - Any valid reviewer or the PR author doing `/lgtm cancel` will reset the count of `lgtm`. 
- Phase 2: Humans approve the PR
  - The PR **author** `/assign`'s all suggested **approvers** to the PR, and optionally notifies
    them (eg: "pinging @foo for approval")
  - Only people listed in the relevant OWNERS files, either directly or through an alias, as [described
    above](#owners_aliases), can act as **approvers**, including the individual who opened the PR.
  - **Approvers** look for holistic acceptance criteria, including dependencies with other features,
    forwards/backwards compatibility, API and flag definitions, etc
  - If the code changes look good to them, an **approver** types `/approve` in a PR comment or
    review; if they change their mind, they `/approve cancel`
  - [prow](https://prow.tidb.net) ([@ti-chi-bot](https://github.com/apps/ti-chi-bot)) updates its
    comment in the PR to indicate which **approvers** still need to approve
  - Once all **approvers** (one from each of the previously identified OWNERS files) have approved,
    [prow](https://prow.tidb.net) ([@ti-chi-bot](https://github.com/apps/ti-chi-bot)) applies an
    `approved` label
- Phase 3: Automation merges the PR:
  - If all of the following are true:
    - All required labels are present (eg: `lgtm`, `approved`)
    - Any blocking labels are missing (eg: there is no `do-not-merge/hold`, `needs-rebase`)
  - And if any of the following are true:
    - there are no presubmit prow jobs configured for this repo
    - there are presubmit prow jobs configured for this repo, and they all pass after automatically
      being re-run one last time
  - Then the PR will automatically be merged

> Modified based on [kubernetes community review process](https://github.com/kubernetes/community/blob/master/contributors/guide/owners.md#code-review-using-owners-files).

## Recommended configuration items 

### It is recommended to use Squash mode to merge code

In terms of merging, we still recommend using GitHub's Squash mode for merging, because this is the current tradition of the TiDB community. Everyone will create a large number of commits in the PR, and then automatically perform Squash through GitHub when merging. At present, our ti-community-merge is also designed to serve the Squash mode. **If you do not use the Squash mode, then you need to be responsible for rebase or squash commits in the PR, which will invalidate our function of storing and submitting the hash (details See Q&A), eventually causing status/can-merge to automatically cancel** because there are new commits. So we strongly recommend that you use Squash mode for collaboration. 

### If the CI task of the repository is triggered by prow, you need to trun off the "Require branches to be up to date before merging branch protection" option. 

If it is a CI task triggered by prow, the pre-merge with the base has been performed in the checkout link before subsequent construction steps . 

## Q&A 

### Why does my own rebase or squash commit cause `lgtm` to be removed? 

**Because we are currently storing the hash of the last commit in your PR when you tagged `lgtm`**. When you rebase the PR, the entire hash will change, so the label will be automatically removed. When you squash PR yourself, because we store the hash of the last submission instead of the hash of the first submission, this will still lead to automatic removing.
