#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
for tool in yq kubectl helm kubeconform; do
  command -v "$tool" >/dev/null || { echo "Missing required tool: $tool" >&2; exit 1; }
done
output=$(mktemp -d)
trap 'rm -rf "$output"' EXIT
while IFS= read -r file; do
  yq eval '.' "$file" >/dev/null
done < <(find bootstrap catalog clusters .github -type f -name '*.yaml' | sort)
for file in mkdocs.yml catalog-info.yaml; do yq eval '.' "$file" >/dev/null; done
count=0
while IFS= read -r file; do
  kubectl kustomize "$(dirname "$file")" > "$output/rendered.yaml"
  kubeconform -kubernetes-version 1.32.0 -strict -summary -skip Application,AppProject,GatewayClass,Gateway,HTTPRoute "$output/rendered.yaml"
  count=$((count + 1))
done < <(find bootstrap catalog clusters -name kustomization.yaml | sort)
while IFS= read -r file; do
  source_path=$(yq -r '.spec.source.path' "$file")
  test -f "$source_path/kustomization.yaml"
  root_name=$(yq -r '.metadata.name' "$file")
  kubectl kustomize "$source_path" > "$output/root.yaml"
  while IFS= read -r app_name; do
    APP_NAME="$app_name" yq 'select(.kind == "Application" and .metadata.name == strenv(APP_NAME))' \
      "$output/root.yaml" > "$output/app-$root_name-$app_name.yaml"
  done < <(yq -N -r 'select(.kind == "Application") | .metadata.name' "$output/root.yaml")
done < <(find bootstrap/roots -name '*.yaml' | sort)
while IFS= read -r file; do
  cp "$file" "$output/app-example-$(basename "$file")"
done < <(find bootstrap/examples -path '*/argocd/applications/*.yaml' | sort)
while IFS= read -r file; do
  source_path=$(yq -r '.spec.source.path // ""' "$file")
  if [[ -n "$source_path" ]]; then
    test -f "$source_path/kustomization.yaml"
    continue
  fi
  repo=$(yq -r '.spec.sources[0].repoURL' "$file")
  chart=$(yq -r '.spec.sources[0].chart' "$file")
  version=$(yq -r '.spec.sources[0].targetRevision' "$file")
  release=$(yq -r '.spec.sources[0].helm.releaseName' "$file")
  namespace=$(yq -r '.spec.destination.namespace' "$file")
  values=()
  while IFS= read -r value; do
    value=${value#\$values/}
    test -f "$value"
    values+=(-f "$value")
  done < <(yq -r '.spec.sources[0].helm.valueFiles[]' "$file")
  chart_args=("$chart" --repo "$repo")
  if [[ "$repo" != https://* ]]; then chart_args=("oci://$repo/$chart"); fi
  helm template "$release" "${chart_args[@]}" --version "$version" --namespace "$namespace" \
    --kube-version 1.32.0 "${values[@]}" > "$output/helm.yaml"
  test -s "$output/helm.yaml"
  echo "Rendered $file"
done < <(find "$output" -name 'app-*.yaml' | sort)
echo "Validated $count Kustomizations and all Application source paths and Helm values."
