#!/usr/bin/env bash

echo "Build label_sync..." >&2

# Add -mod=mod flag for go build because the default behavior of go build have changed in go 1.16.
# Refer: https://github.com/golang/go/issues/44212#issuecomment-776937327
go build -mod=mod -o tools/bin/label_sync k8s.io/test-infra/label_sync

echo "Validating labels..." >&2

label_sync=$PWD/tools/bin/label_sync

cd configs/prow-dev/config || exit

"$label_sync" \
  --config=labels.yaml \
  --action=docs \
  --docs-template=labels.md.tmpl \
  --docs-output=labels.md.expected

DIFF=$(diff labels.md labels.md.expected || true)
if [[ -n "$DIFF" ]]; then
  echo "< unexpected" >&2
  echo "> missing" >&2
  echo "${DIFF}" >&2
  echo "" >&2
  echo "ERROR: labels.md out of date. Fix with scripts/update-labels.sh" >&2
  exit 1
fi
