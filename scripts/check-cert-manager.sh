#!/usr/bin/env bash
# Check the effective chart, not just the input values, for bootstrap regressions.
set -euo pipefail
chart=$1
output=$2
issuer=catalog/platform/cert-manager/issuers
test "$(yq '[select(.kind == "CustomResourceDefinition")] | length' "$chart" | awk '{s+=$1} END {print s}')" = 6
yq -e 'select(.kind == "Job") | .metadata.annotations."argocd.argoproj.io/hook" == "Sync" and .metadata.annotations."argocd.argoproj.io/sync-wave" == "1"' "$chart" >/dev/null
test -z "$(yq -r 'select(.metadata.annotations."helm.sh/hook" != null) | .metadata.name' "$chart")"
kubectl kustomize "$issuer" > "$output/issuer.yaml"
yq -e '.metadata.annotations."argocd.argoproj.io/sync-wave" == "2" and .spec.acme.solvers[0].http01.ingress.ingressClassName == "webapprouting.kubernetes.azure.com"' "$output/issuer.yaml" >/dev/null
yq -o=json 'select(.kind == "CustomResourceDefinition" and .metadata.name == "clusterissuers.cert-manager.io") | .spec.versions[] | select(.name == "v1") | .schema.openAPIV3Schema' "$chart" > "$output/clusterissuer.json"
kubeconform -strict -summary -schema-location "$output/clusterissuer.json" "$output/issuer.yaml"
# The standard Kubernetes schema catalog does not publish CRD schemas.
kubeconform -kubernetes-version 1.32.0 -strict -summary -skip CustomResourceDefinition "$chart"
# Every rendered resource must be allowed by the dedicated AppProject.
project=catalog/platform/cert-manager/project.yaml
while IFS=$'\t' read -r api kind namespace; do
  group=${api%/*}
  [[ "$api" == v1 ]] && group=''
  if [[ -n "$namespace" ]]; then
    NAMESPACE="$namespace" yq -e '.spec.destinations[] | select(.namespace == strenv(NAMESPACE))' "$project" >/dev/null
  else
    GROUP="$group" KIND="$kind" yq -e '.spec.clusterResourceWhitelist[] | select(.group == strenv(GROUP) and .kind == strenv(KIND))' "$project" >/dev/null
  fi
done < <(yq -N -r '[.apiVersion, .kind, (.metadata.namespace // "")] | @tsv' "$chart" "$output/issuer.yaml")
