#!/bin/bash

set -o errexit

CGO_ENABLED=0 go build -o webhook main.go
docker build . -f webhook.dockerfile -t gcr.io/stclair-k8s-ext/webhook-test:latest
docker push gcr.io/stclair-k8s-ext/webhook-test:latest
