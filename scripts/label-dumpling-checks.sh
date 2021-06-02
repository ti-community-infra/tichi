#!/bin/bash

# workdir
cd ./tools/label-dumpling/ || exit

# add components
rustup component add rustfmt
rustup component add clippy

# checks
cargo fmt --all -- --check
cargo check --all --all-targets
cargo clippy --all --all-targets -- -D warnings
