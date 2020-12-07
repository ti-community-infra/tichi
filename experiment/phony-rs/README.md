# Phony

`phony-rs` sends fake GitHub webhooks.

## Used to test the GitHub event manager or plugin
`phony-rs` is most commonly used for testing [hook](https://github.com/kubernetes/test-infra/tree/master/prow/hook) and its [plugins](https://github.com/kubernetes/test-infra/tree/master/prow/plugins), but can be used for testing any externally exposed service configured to receive GitHub events (external plugins).

## Usage
Once you have a running server that manages github webhook events, you can `cargo run` and sent events to that server. 

A list of supported events can be found in the [GitHub API Docs](https://developer.github.com/v3/activity/events/types/).