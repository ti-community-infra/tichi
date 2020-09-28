SHELL := /bin/bash

prow-namespace:
	kubectl apply -f cluster/prow_namespace.yaml

test-pods-namespace:
	kubectl apply -f cluster/test-pods_namespace.yaml

oauth:
	kubectl apply -f cluster/oauth-token.yaml

hmac:
	kubectl apply -f cluster/hmac-token.yaml

prow: prow-namespace test-pods-namespace oauth hmac
	kubectl apply -f cluster

plugins:
	kubectl apply -f cluster/plugin.yaml

configs:
	kubectl apply -f cluster/config.yaml

clean:
	rm -f cluster/oauth-token.yaml
	rm -f cluster/hmac-token.yaml