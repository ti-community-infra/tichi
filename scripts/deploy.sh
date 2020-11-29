#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

if [[ -n "${GOOGLE_APPLICATION_CREDENTIALS:-}" ]]; then
  echo "Detected GOOGLE_APPLICATION_CREDENTIALS, activating..." >&2
  gcloud auth activate-service-account --key-file="$GOOGLE_APPLICATION_CREDENTIALS"
fi

case "${1:-}" in
"--confirm")
  shift
  ;;
*)
  read -p "Deploy prow to dev [no]: " confirm
  if [[ "${confirm}" != y* ]]; then
    exit 1
  fi
  ;;
esac

# See https://misc.flogisoft.com/bash/tip_colors_and_formatting
color-green() { # Green
  echo -e "\x1B[1;32m${*}\x1B[0m"
}

color-context() { # Bold blue
  echo -e "\x1B[1;34m${*}\x1B[0m"
}

color-missing() { # Yellow
  echo -e "\x1B[1;33m${*}\x1B[0m"
}

ensure-context() {
  local proj=$1
  local zone=$2
  local cluster=$3
  local context="gke_${proj}_${zone}_${cluster}"
  echo -n " $(color-context "$context")"
  kubectl config get-contexts "$context" &> /dev/null && return 0
  echo ": $(color-missing MISSING), getting credentials..."
  gcloud container clusters get-credentials --project="$proj" --zone="$zone" "$cluster"
  kubectl config get-contexts "$context" > /dev/null
  echo -n "Ensuring contexts exist:"
}

echo -n "Ensuring dev context exist:"
current_context=$(kubectl config current-context 2>/dev/null || true)
restore-context() {
  if [[ -n "$current_context" ]]; then
    kubectl config set-context "$current_context"
  fi
}
trap restore-context EXIT
ensure-context pingcap-testing-account us-central1-c prow-dev
echo " $(color-green 'done'), Deploying prow..."
for s in {5..1}; do
    echo -n $'\r'"in $s..."
    sleep 1s
done

# Apply
pwd
cd ../configs/prow-dev/
make cluster
echo "$(color-green 'SUCCESS'), Deployed"
