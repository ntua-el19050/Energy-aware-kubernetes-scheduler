#!/bin/bash

# 🧠 Port mappings
# 9090 → Prometheus

echo "🔌 Starting port-forwards..."

# Prometheus
microk8s kubectl port-forward -n observability prometheus-kube-prom-stack-kube-prome-prometheus-0 9090:9090 &

echo "✅ All port-forwards active. Press Ctrl+C to stop."
wait

