#!/bin/bash

set -o errexit

openssl req -new -newkey rsa:1024 -x509 -sha256 -days 365 -nodes -out webhook.crt -keyout webhook.key

kubectl create secret tls webhook-certs --cert=webhook.crt --key=webhook.key  -o yaml > webhook-secret.yaml

