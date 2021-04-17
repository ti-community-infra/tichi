# release-note

## Design Background

When large open source software releases are made, a release note is usually provided to describe the major changes and updates in the release. To better collect this information, the TiDB community asks contributors to add release notes to their PRs for organization and use at release time.

[release-note](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/releasenote) checks if the release note is added to the body of the PR according to the standard format and adds the corresponding label to mark the PR.

## Design

This plugin was designed and developed by the Kubernetes community to detect if a release note in the following format has been added to the Body of a PR:

```go
noteMatcherRE = regexp.MustCompile(`(?s)(?:Release note\*\*:\s*(?:<!--[^<>]*-->\s*)?` + "```(?:release-note)?|```release-note)(.+?)```")
```

We recommend completing the release note in this format:

```release-note
Some release note.
```

The advantage of using markdown code block to organize the release note is that the format is simple and easy to read after markdown rendering. **When we fill in the release note in this format, the release-note plugin adds a `release-note` label to the PR to mark that the release note have been properly added.**

In addition, considering that some code refactorings or non-code-related changes may not require a release note, we can also use None to indicate that there is no release note:

```release-note
None
```

Same format as above, but with None in the block to indicate that no release notes are needed. **When we put None in the release note, release-note plugin adds `release-note-none` label to the PR to mark the PR as not requiring a release note.**

If you find it complicated to fill in this content for a PR that doesn't require a release note, you can also add the `release-note-none` label directly using the `/release-note-none` command.

Finally, if the release-note plugin does not detect any release-note code blocks and you do not mark the PR as not requiring a release note with the command, then release-note plugin will add the `do-not-merge/release-note-label- needed` label to the PR to prevent merging of the PR.

## Parameter Configuration 

No configuration

## 参考文档

- [command help](https://prow.tidb.io/command-help#release_note_none)
- [code](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/releasenote)