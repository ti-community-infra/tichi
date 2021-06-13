# Contributing

We're so excited you're interested in helping with TiChi! We are happy to help you get started, even if you don't have
any previous open-source experience.

## New to Open Source?

1. Take a look
   at [How to Contribute to an Open Source Project on GitHub](https://egghead.io/courses/how-to-contribute-to-an-open-source-project-on-github)
2. Go thorough the [TiChi Code of Conduct](https://github.com/ti-community-infra/tichi/blob/master/CODE_OF_CONDUCT.md)

## Where to ask Questions?

1. Check our [Github Issues](https://github.com/ti-community-infra/tichi/issues) to see if someone has already answered
   your question.
2. Join our community on [Slack](https://slack.tidb.io/invite?team=tidb-community&channel=sig-community-infra) and feel
   free to ask us your questions

As you gain experience with TiChi, please help answer other people's questions!

## What to Work On?

You can get started by taking a look at our [Github Issues](https://github.com/ti-community-infra/tichi/issues)  
If you find one that looks interesting and no one else is already working on it, comment in the issue that you are going
to work on it.

Please ask as many questions as you need, either directly in the issue or
on [Slack](https://slack.tidb.io/invite?team=tidb-community&channel=sig-community-infra). We're happy to help!

### Contributions that are ALWAYS welcome

1. More tests
2. Improved messages
3. Documentation improvement and translation

## Development Setup

### Prerequisites

- OS: Linux or macOS or Windows
- Golang: 1.16
- IDE: [GoLand](https://www.jetbrains.com/go/)(recommended) or equivalent IDE

### Familiarize yourself with TiChi

TiChi uses Prow as the basic framework, so understanding Prow means understanding TiChi.

1. [Prow](https://github.com/kubernetes/test-infra/tree/master/prow)
2. [How Prow works](https://www.youtube.com/watch?v=qQvoImxHydk)
3. [How to build a Prow plugin](https://github.com/ti-community-infra/tichi/pull/425)

### Project Setup

1. Fork the [TiChi](https://github.com/ti-community-infra/tichi) repository
2. `git clone https://github.com/<YOUR_GITHUB_LOGIN>/tichi`
3. `cd tichi`
4. `make dev` to get dependencies and run tests

## Modifying code

1. Open `tichi` in your IDE
2. Modifying the code

## Testing

1. Go to the `tichi` root directory
2. Run all tests through the `make test` command
3. If all tests pass the terminal should display

## Pull Request

1. Before submitting a pull request make sure all tests have passed
2. Reference the relevant issue or pull request and give a clear description of changes/features added when submitting a
   pull request

## TiChi Community

If you have any questions or would like to get more involved in the TiChi community you can check out:

- [Github Issues](https://github.com/ti-community-infra/tichi/issues)
- [Slack](https://slack.tidb.io/invite?team=tidb-community&channel=sig-community-infra)

Additional resources you might find useful:

- [TiChi documentation](https://book.prow.tidb.io/#/)
