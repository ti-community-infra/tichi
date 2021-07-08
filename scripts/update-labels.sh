#!/usr/bin/env bash

echo "Build label_sync..." >&2

# Add -mod=mod flag for go build because the default behavior of go build have changed in go 1.16.
# Refer: https://github.com/golang/go/issues/44212#issuecomment-776937327
go build -mod=mod -o tools/bin/label_sync k8s.io/test-infra/label_sync

label_sync=$PWD/tools/bin/label_sync

cd configs/prow-dev/config || exit

# Generate labels.
"$label_sync" \
  --config=labels.yaml \
  --action=docs \
  --docs-template=labels.md.tmpl \
  --docs-output=labels.md
