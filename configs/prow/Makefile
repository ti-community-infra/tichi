SHELL := /bin/bash

prow-namespace:
	kubectl apply -f cluster/prow_namespace.yaml

test-pods-namespace:
	kubectl apply -f cluster/test-pods_namespace.yaml

github:
	kubectl apply -f cluster/github-token.yaml

oauth:
	kubectl apply -f cluster/oauth-token.yaml

oauth-cookie:
	kubectl apply -f cluster/oauth-cookie.yaml

hmac:
	kubectl apply -f cluster/hmac-token.yaml

gcs:
	kubectl apply -f cluster/gcs_secrets.yaml

prow: prow-namespace test-pods-namespace github oauth oauth-cookie hmac gcs
	kubectl apply -f cluster

plugins: config/plugins.yaml
	kubectl create configmap plugins -n prow --from-file=plugins.yaml=config/plugins.yaml --dry-run -o yaml | kubectl replace configmap plugins -n prow -f -

configs: config/config.yaml
	kubectl create configmap config -n prow --from-file=config.yaml=config/config.yaml --dry-run -o yaml | kubectl replace configmap config -n prow -f -

external-configs:
	kubectl apply -f config/external_plugins_config.yaml

clean:
	rm -f cluster/github-token.yaml
	rm -f cluster/oauth-token.yaml
	rm -f cluster/oauth-cookie.yaml
	rm -f cluster/hmac-token.yaml
	rm -f cluster/gcs-token.yaml