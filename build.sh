#!/bin/bash
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64
go build
docker build -t harbor.finmas.co.id/devops/dns-grpc:latest .
docker push   harbor.finmas.co.id/devops/dns-grpc:latest
