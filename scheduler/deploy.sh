#!/bin/bash

set -e  # Exit on error

echo "🔨 Building Docker image..."
docker build -t dockerhub_user/custom-scheduler:latest .

echo "📤 Pushing Docker image to Docker Hub..."
docker push dockerhub_user/custom-scheduler:latest

echo "⚙️ Creating scheduler ConfigMap..."
microk8s kubectl create configmap my-scheduler-config \
  --from-file=kube-scheduler-config.yaml=/home/ubuntu/scheduler/kube-scheduler-config.yaml \
  -n kube-system

echo "🚀 Deploying custom scheduler..."
microk8s kubectl apply -f scheduler-deployment.yaml

echo "✅ Done!"

