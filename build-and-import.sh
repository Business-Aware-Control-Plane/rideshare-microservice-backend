#!/usr/bin/env bash

set -e

CLUSTER_NAME="${1:-fyp-cluster}"

echo "=========================================="
echo " Building Go Binaries (Linux/amd64)"
echo "=========================================="
mkdir -p build

echo "Compiling api-gateway..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/api-gateway ./services/api-gateway

echo "Compiling trip-service..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/trip-service ./services/trip-service/cmd/main.go

echo "Compiling driver-service..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/driver-service ./services/driver-service

echo "Compiling payment-service..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/payment-service ./services/payment-service/cmd/main.go

echo "=========================================="
echo " Building Docker Images"
echo "=========================================="

echo "Building ride-sharing/api-gateway:latest..."
docker build -t ride-sharing/api-gateway:latest -f ./infra/development/docker/api-gateway.Dockerfile .

echo "Building ride-sharing/trip-service:latest..."
docker build -t ride-sharing/trip-service:latest -f ./infra/development/docker/trip-service.Dockerfile .

echo "Building ride-sharing/driver-service:latest..."
docker build -t ride-sharing/driver-service:latest -f ./infra/development/docker/driver-service.Dockerfile .

echo "Building ride-sharing/payment-service:latest..."
docker build -t ride-sharing/payment-service:latest -f ./infra/development/docker/payment-service.Dockerfile .

echo "Building ride-sharing/web:latest..."
docker build -t ride-sharing/web:latest -f ./infra/development/docker/web.Dockerfile .

echo "=========================================="
echo " Importing Images into k3d cluster '${CLUSTER_NAME}'"
echo "=========================================="
k3d image import \
  ride-sharing/api-gateway:latest \
  ride-sharing/trip-service:latest \
  ride-sharing/driver-service:latest \
  ride-sharing/payment-service:latest \
  ride-sharing/web:latest \
  -c "${CLUSTER_NAME}"

echo "=========================================="
echo " Done! Triggering pod refresh in k3d..."
echo "=========================================="
if kubectl get secret postgres-cluster-app -n platform &>/dev/null; then
  echo "Syncing CloudNativePG secret from 'platform' to 'rideshare' namespace..."
  POSTGRES_URI=$(kubectl get secret postgres-cluster-app -n platform -o jsonpath='{.data.uri}' | base64 -d)
  kubectl create secret generic postgres-cluster-app \
    --from-literal=uri="${POSTGRES_URI}?sslmode=disable" \
    -n rideshare --dry-run=client -o yaml | kubectl apply -f -
fi

kubectl rollout restart deployment -n rideshare

echo "Check pod status with: kubectl get pods -n rideshare"
