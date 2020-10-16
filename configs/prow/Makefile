SHELL := /bin/bash

prow-namespace:
	kubectl apply -f cluster/prow_namespace.yaml

test-pods-namespace:
	kubectl apply -f cluster/test-pods_namespace.yaml

oauth:
	kubectl apply -f cluster/oauth-token.yaml

hmac:
	kubectl apply -f cluster/hmac-token.yaml

gcs:
	kubectl apply -f cluster/gcs_secrets.yaml

prow: prow-namespace test-pods-namespace oauth hmac gcs
	kubectl apply -f config
	kubectl apply -f cluster

plugins:
	kubectl apply -f config/plugin.yaml

configs:
	kubectl apply -f config/config.yaml

external-configs:
	kubectl apply -f config/external_plugins_config.yaml

clean:
	rm -f cluster/oauth-token.yaml
	rm -f cluster/hmac-token.yaml