# [Tide](https://github.com/kubernetes/test-infra/tree/master/prow/tide)

Tide is a core component of Prow that focuses on managing the GitHub PR pool with a few given conditions. It will automatically re-detect PRs that meet the conditions and automatically merge them when they pass the test.

It has the following features:
- Automatically run batch tests and merge multiple PRs together when possible. (This feature is disabled if you do not use Prow's CI)
- Make sure to test the PR against the most recent base branch commit before allowing the PR to be merged. (This feature is disabled if you do not use Prow's CI)
- Maintain a GitHub status context that indicates whether each PR is in the PR pool or what requirements are missing. (This is a status similar to what other CIs report in their PRs, and the current status of the PR is specified in the message for that status)
- Support for blocking PR merges to a single branch or an entire repository using a special GitHub Label.
- Prometheus metrics.
- Supports having "optional" state contexts that are not mandatory for merging.
- Provides real-time data about the current PR pool and attempted merge history, which can be displayed in [Deck](https://github.com/kubernetes/test-infra/tree/master/prow/cmd/deck), [Tide Dashboard](https://prow.tidb.io/tide), [PR Status](https://prow.tidb.io/pr), and [Tide History](https://prow.tidb.io/tide-history).
- Effectively scales so that a single instance with a single bot token can provide merge automation for dozens of organizations and repositories that meet the merge criteria. Each different `org/repo:branch` combination defines a merge pool that does not interfere with each other, so merging only affects other PRs in the same branch.
- Provides configurable merge modes (`merge`, `squash`, or `rebase`).

## Tide in the TiDB Community

### Tide's periodic scans

As the name Tide implies, Tide does not immediately trigger a check when a PR is opened or a new commit is committed. Instead, it takes the strategy of periodically (every minute or two) running a full scan of all repositories hosted by Tide to check which open PRs meet the merge conditions, and if they do, they are sent to the merge pool (queue) for merge.

**So, if you find no tide context in the PR's checks, or it prompts "Waiting for status to be reported", please wait for a while. **

We can control the period of the full scan with the ``tide.sync_period`` configuration, an example configuration is as follows.

```yaml
tide:
  sync_period: 2m
```

### Configuring PR merge rules

We can configure the merge rules for PRs of org, repo or branch by configuring `queries` or `context_options`.

Please refer to the [Configuring Tide](https://github.com/kubernetes/test-infra/blob/master/prow/cmd/tide/config.md#configuring-tide) documentation for detailed configuration.

Example configuration:

```yaml
tide:
  queries:
  - repos:
      - pingcap/community
    includedBranches:
      - master # Only the PRs on the master branch will be merged
    labels:
      - status/can-merge # PRs from this repository will only be merged if they are labeled as status/can-merge.
    missingLabels:
      # PRs from this repository will not be merged if they have the following labels
      - do-not-merge
      - do-not-merge/hold
      - do-not-merge/work-in-progress
      - needs-rebase
    reviewApprovedRequired: true # PRs must meet the review criteria on GitHub.

  context_options:
    orgs:
      pingcap:
        repos:
          community:
            # Require that all PRs for all branches of this repository must pass the CI of license/cla.
            required-contexts:
              - "license/cla"
            branches:
              master:
                # Require all PRs for the master of this repository to pass the CI of Sig Info File Format.
                required-contexts:
                  - "Sig Info File Format"
```

### Check the working status of Tide

Get the PR status from these places.

#### PR Status Context

Contributor can check the status of the PR in the CI Status Context under the PR, e.g. what status checks must be passed before the merge, what required labels are missing from the PR, or what labels are not allowed on the PR. If all conditions are passed, the PR will go to the merge pool (queue) and wait to be merged (prompt: `In merge pool.`).

![PR Status Context](https://user-images.githubusercontent.com/29879298/98230629-54037400-1f96-11eb-8a9c-1144905fbbd5.png)

Click on the "Details" to jump to the [Tide Dashboard](https://prow.tidb.io/tide).

#### PR Status Page

Contributors can view the status of their PR submissions on the [PR status](https://prow.tidb.io/pr) page, where each of your PRs will have a card showing the test results and merge requirements. (Recommended)

#### Tide Dashboard

In the [Tide Dashboard](https://prow.tidb.io/tide), the status of each merge pool is displayed, allowing you to see what Tide is currently doing and where the PR is in the retest queue. If a PR in a merge pool (queue) has been in the merge state for a long time, then

### Tide's merge method

GitHub provides us with merge / [squash](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a- pull-request/about-pull-request-merges#squash-and-merge-your-pull-request-commits) / [rebase](https://docs.github.com/en/pull- requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/about-pull-request-merges#rebase-and-merge- your-pull-request-commits) three types of Pull Requests are merged. We can set Tide's default PR merge method via the `tide.merge_method.<org_or_repo_name>` configuration, or we can specify the merge method for a given PR by adding a label.

The associated label is determined by the `squash_label`, `rebase_label`, and `merge_label` configuration items.

The TiDB community uses `tide/merge-method-squash`, `tide/merge-method-rebase`, and `tide/merge-method-merge` as the labels that define how Tide merges PRs.

Example configuration.

```yaml
tide:
  merge_method:
    pingcap/community: squash # The default merge method for this repository is squash.

  squash_label: tide/merge-method-squash
  rebase_label: tide/merge-method-rebase
  merge_label: tide/merge-method-merge
```

Note: Before using other merge methods, you need to make sure the merge method is allowed in the repository settings. **The configuration of the repository will not be synced to Tide immediately after the changes are made, Tide will go and get the latest repository settings every hour**.

![Merge Button Setting](https://user-images.githubusercontent.com/5086433/151337189-29ae600b-2c1d-4bb3-bf71-b852d556f3cf.png)


### Custom Commit Message Template

When merging PRs manually, GitHub provides us with a page form to fill in the message title and message body parts of the final commit that results from the PR merge.

![Commit Message Form](https://user-images.githubusercontent.com/5086433/151338288-63da93b5-cd35-4622-842f-7f16fcc299f7.png)

Now we use Tide to help us automate the PR merge, Tide provides us with [`commit_message_template`](https://github.com/ti-community-infra/configs/blob/main/prow/config/) config.yaml#:~:text=merge_commit_template) to configure a template for customizing the commit message title and commit message body.

```yaml
tide:
  merge_commit_template:
    ti-community-infra/test-dev:
      title: "{{ .Title }} (#{{ .Number }})"
      body: |
        {{ .Body }}
```

We can define the commit message template for the entire org repos or for a single repo, e.g. for the above configuration, the contents of the `title` and `body` configurations will be processed by go [text template](https://pkg.go.dev/text/template ) will be processed by go [text template](https://pkg.go.dev/k8s.io/test-infra/prow/tide#PullRequest) and populated with data related to [Pull Request](https://pkg.go.dev/k8s.io/test-infra/prow/tide#PullRequest) to generate the final commit message.

If the `title` or `body` configuration is empty, GitHub will use the default commit message when merging the PR (e.g., [merge-message-for-a-squash-merge](https://docs.github.com/en/pull- requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/about-pull-request-merges#merge-message-for-a squash-merge) ) as the title and body of the commit message. If you wish to fill in an empty commit message body, you can set the `body` configuration to the empty string `" "`.

#### utility functions

To extend the capabilities of the commit message template, the TiDB community has included in the [Custom Tide component](https://github.com/ti-community-infra/test-infra/tree/ti-community-custom/prow/tide) some utility functions for the text template, the TiDB community has injected a number of useful utility functions into the [custom Tide component]().

##### `.ExtractContent <regexp> <source_text>`

The `.ExtractContent` function extracts text content from a regular expression, and when the regular expression contains a named group named `content`, it returns only the content matched by the [named group](https://pkg.go.dev/regexp#Regexp.SubexpNames). If not, it returns the content matched by the entire regular expression.

Example configuration:

```yaml
tide:
  merge_commit_template:
    ti-community-infra/test-dev:
      title: "{{ .Title }} (#{{ .Number }})"
      body: |
        {{- $body := print .}
        {{- $description := .ExtractContent "(?i)\x60\x60\x60commit-message(?P<content>[\w|\\\W]+)\x60\x60\x60" $body -}}
        {{- if $description -}}{{- "\n\n" -}}{{- end -}}
        {{- $description -}}
```

##### `.NormalizeIssueNumbers <source_text>`

The `.NormalizeIssueNumbers` function extracts the issue number from the text content list, formats it, and returns an array of issue number objects.

Linked issue numbers must be prefixed with a GitHub-supported keyword or the agreed-upon `ref` keyword.

Example configuration:

```yaml
tide:
  merge_commit_template:
    ti-community-infra/test-dev:
      title: "{{ .Title }} (#{{ .Number }})"
      body: |
        {{- $body := print .Body -}}
        {{- $issueNumberLine := .ExtractContent "(?im)^Issue Number:. +" $body -}}
        {{- $numbers := .NormalizeIssueNumbers $issueNumberLine -}}
        {{- if $numbers -}}
          {{- range $index, $number := $numbers -}}
            {{- if $index }}, {{ end -}}
            {{- .AssociatePrefix }} {{ .Org -}}/{{- .Repo -}}#{{- .Number -}}
          {{- end -}}
        {{- else -}}
          {{- " " -}}
        {{- end -}}
```

In the above configuration, if there is a line `Issue Number: close #123, ref #456` in the PR body, the template will return the text `close org/repo#123, ref org/repo#456`.

##### `.NormalizeSignedOffBy`

This function formats the signature (`Signed-off-by: `) information in PR commits and returns an array of signed-author objects.

Some open source repositories use the [DCO protocol](https://wiki.linuxfoundation.org/dco) instead of the [CLA protocol](https://en.wikipedia.org/wiki/Contributor_License_Agreement), which The former requires Contributors to indicate their acceptance of the protocol by filling in the `Signned-off-by: ` line in the commit.

When using the squash method to merge PRs, multiple commits in a PR are merged into a single squashed commit before being merged into a base branch. To simplify the message of a squashed commit, we can use the `.NormalizeSignedOffBy` function to de-duplicate and merge the `Signed-off-by:` messages of multiple commits.

Example configuration:

```yaml
tide:
  merge_commit_template:
    ti-community-infra/test-dev:
      title: "{{ .Title }} (#{{ .Number }})"
      body: |
        {{- $signedAuthors := .NormalizeSignedOffBy -}}
        {{- if $signedAuthors -}}{{- "\n\n" -}}{{- end -}}
        {{- range $index, $author := $signedAuthors -}}
        {{- if $index -}}{{- "\n" -}}{{- end -}}
        {{- "Signed-off-by:" }} {{ .Name }} <{{- .Email -}}>
        {{- end -}}
```

##### `.NormalizeCoAuthorBy`

This function formats the information about the co-author of a PR, returning an array of co-author objects.

A PR may have multiple commits from different authors, and the same commit may have multiple authors (GitHub defines that you can use [`Co-authored-by:`](https://docs.github.com/en/pull-) in the commit message) requests/committing-changes-to-your-project/creating-and-editing-commits/creating-a-commit-with-multiple-authors) line in the commit message to declare the commit's co-authors).

When we use the squash method to merge PRs, the multiple commits in the PR are first merged into a new squashed commit and then into the base branch. NormalizeCoAuthorBy` function to de-duplicate and merge the author information of multiple commits.

**This function considers all commit authors or co-authors in the PR other than the PR author to be the co-author of the squashed commit.**

Example configuration:

```yaml
tide:
  merge_commit_template:
    ti-community-infra/test-dev:
      title: "{{ .Title }} (#{{ .Number }})"
      body: |
        {{- $coAuthors := .NormalizeCoAuthorBy -}}
        {{- if $coAuthors -}}{{- "\n\n" -}}{{- end -}}
        {{- range $index, $author := $coAuthors -}}
          {{- if $index -}}{{- "\n" -}}{{- end -}}
          {{- "Co-authored-by:" }} {{ .Name }} <{{- .Email -}}>
        {{- end -}}
```

### PR merge with CI test

Tide works mostly fine in the TiDB community, but there's still a tricky issue (**other communities haven't solved it yet either**):

- PR1: Rename bifurcate() to bifurcateCrab()
- PR2: Use bifurcate()
  
In this case, both PRs will be tested with the current master as the base branch, and both PRs will pass. However, once PR1 is merged into the master branch first, and the second PR is merged (because the test also passes), it causes a master error that `bifurcate` is not found.

I will describe how to solve this problem in the recommended PR workflow.

**The Kubernetes community does not currently have this issue because if you use Prow's CI system Tide will automatically check out to the latest base for testing**.

### Q&A

#### Is my PR in the merge queue?

If the status of Tide is successful (green) then it is already in the merge pool, if it is unsuccessful (yellow) then it will not be in the merge pool.

#### Why is my PR not in the merge queue?

If you have just updated your PR, please wait a bit to give Tide time to detect it (one minute by default).

Whether your PR is in the queue or not is determined by the following two requirements:
- Check that your PR label meets the requirements.
- Check that the required CIs have all passed.

#### Why is the status of Tide still Penning even though PR is merged?

This is because in some cases it may detect that the PR has been met and merge it, but then merge it before it has a chance to update the status to GitHub.

#### If one of the tests in my PR fails, do I need to run them all again, or just one?

You only need to re-run the failed test.

## Reference Documents

- [Maintainer's Guide to Tide](https://github.com/kubernetes/test-infra/blob/master/prow/cmd/tide/maintainers.md)
- [bors-ng](https://github.com/bors-ng/bors-ng)


