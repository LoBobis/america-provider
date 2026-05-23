#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KUBECONFIG="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"
TIMEOUT="${E2E_TIMEOUT:-180}"
PROVIDER_IMAGE="${PROVIDER_IMAGE:?PROVIDER_IMAGE must be set}"
MOCK_AMERICA_IMAGE="${MOCK_AMERICA_IMAGE:?MOCK_AMERICA_IMAGE must be set}"

export KUBECONFIG

echo "=== Waiting for k3s to be ready ==="
until kubectl get nodes 2>/dev/null | grep -q " Ready"; do
  echo "Waiting for k3s node..."
  sleep 5
done
echo "k3s is ready"

echo "=== Installing CRDs ==="
kubectl apply -R -f "${SCRIPT_DIR}/../../package/crds/"
kubectl wait --for=condition=Established --all crd --timeout=60s

echo "=== Deploying mock America API ==="
sed "s|\${MOCK_AMERICA_IMAGE}|${MOCK_AMERICA_IMAGE}|g" \
  "${SCRIPT_DIR}/manifests/mock-america.yaml" | kubectl apply -f -

echo "=== Deploying provider ==="
sed "s|\${PROVIDER_IMAGE}|${PROVIDER_IMAGE}|g" \
  "${SCRIPT_DIR}/manifests/provider-deployment.yaml" | kubectl apply -f -

echo "=== Waiting for mock America API ==="
kubectl -n crossplane-system wait --for=condition=Available deployment/mock-america --timeout=120s

echo "=== Waiting for provider ==="
kubectl -n crossplane-system wait --for=condition=Available deployment/provider-america --timeout=120s

echo "=== Creating ProviderConfig ==="
kubectl apply -f "${SCRIPT_DIR}/manifests/provider-config.yaml"
sleep 5

echo "=== Creating test resources ==="
kubectl apply -f "${SCRIPT_DIR}/manifests/test-resources.yaml"

echo "=== Waiting for resources to become Ready ==="
ELAPSED=0
INTERVAL=10
ALL_READY=false

while [ "$ELAPSED" -lt "$TIMEOUT" ]; do
  READY_COUNT=0
  TOTAL=3

  for resource in "topic/test-topic" "datapower/test-datapower" "service/test-service"; do
    KIND=$(echo "$resource" | cut -d/ -f1)
    NAME=$(echo "$resource" | cut -d/ -f2)
    STATUS=$(kubectl get "$KIND" "$NAME" -n default -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "")
    if [ "$STATUS" = "True" ]; then
      READY_COUNT=$((READY_COUNT + 1))
      echo "  $KIND/$NAME: Ready"
    else
      echo "  $KIND/$NAME: Not Ready (status=$STATUS)"
    fi
  done

  if [ "$READY_COUNT" -eq "$TOTAL" ]; then
    ALL_READY=true
    break
  fi

  echo "  ($READY_COUNT/$TOTAL ready, elapsed ${ELAPSED}s)"
  sleep "$INTERVAL"
  ELAPSED=$((ELAPSED + INTERVAL))
done

echo ""
echo "=== Final resource status ==="
kubectl get topic,datapower,service -n default -o wide 2>/dev/null || true

if [ "$ALL_READY" = true ]; then
  echo ""
  echo "=== E2E TESTS PASSED ==="
  exit 0
else
  echo ""
  echo "=== E2E TESTS FAILED ==="
  echo "Not all resources reached Ready state within ${TIMEOUT}s"
  echo ""
  echo "=== Provider logs ==="
  kubectl -n crossplane-system logs deployment/provider-america --tail=50 2>/dev/null || true
  exit 1
fi
