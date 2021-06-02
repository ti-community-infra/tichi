#!/bin/bash

# cd into label dumpling dir.
cd ./tools/label-dumpling/ || exit

# Add fmt and clippy components.
rustup component add rustfmt
rustup component add clippy

# In the CI environment we need to copy the .env file for use by dotenv.
if [[ -z "${CI}" ]]; then
  echo "Not a CI environment, do not copy .env.example to .env"
else
  cp ./tools/label-dumpling/.env.example ./tools/label-dumpling/.env
fi

# checks
cargo fmt --all -- --check
cargo check --all --all-targets
cargo clippy --all --all-targets -- -D warnings
