#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
for tool in yq kubectl helm kubeconform go; do
  command -v "$tool" >/dev/null || { echo "Missing required tool: $tool" >&2; exit 1; }
done
output=$(mktemp -d)
trap 'rm -rf "$output"' EXIT
(cd scripts/render-applications && go test ./... && go build -o "$output/render-applications" .)
while IFS= read -r file; do
  yq eval '.' "$file" >/dev/null
done < <(find bootstrap catalog clusters .github -type f -name '*.yaml' | sort)
for file in mkdocs.yml catalog-info.yaml; do yq eval '.' "$file" >/dev/null; done
count=0
while IFS= read -r file; do
  kubectl kustomize "$(dirname "$file")" > "$output/rendered.yaml"
  kubeconform -kubernetes-version 1.32.0 -strict -summary -skip Application,ApplicationSet,AppProject,GatewayClass,Gateway,HTTPRoute,ClusterIssuer "$output/rendered.yaml"
  count=$((count + 1))
done < <(find bootstrap catalog clusters -name kustomization.yaml | sort)
while IFS= read -r file; do
  source_path=$(yq -r '.spec.source.path' "$file")
  test "$source_path" = "$(dirname "$(dirname "$file")")/argocd"
  test -f "$source_path/kustomization.yaml"
done < <(find clusters -path '*/bootstrap/root.yaml' | sort)
test "$(yq -r '.spec.source.path' bootstrap/local-kind.yaml)" = clusters/local/kind/kind-platform/argocd
while IFS= read -r file; do
  source_path=$(dirname "$file")
  test -f "$source_path/kustomization.yaml"
  root_name=${source_path//\//-}
  kubectl kustomize "$source_path" > "$output/root.yaml"
  cluster=$(dirname "$source_path")
  cp "$output/root.yaml" "$output/applications.yaml"
  while IFS= read -r set_name; do
    SET_NAME="$set_name" yq 'select(.kind == "ApplicationSet" and .metadata.name == strenv(SET_NAME))' \
      "$output/root.yaml" > "$output/applicationset.yaml"
    "$output/render-applications" "$PWD" "$cluster" "$output/applicationset.yaml" > "$output/generated.yaml"
    cat "$output/generated.yaml" >> "$output/applications.yaml"
  done < <(yq -N -r 'select(.kind == "ApplicationSet") | .metadata.name' "$output/root.yaml")
  while IFS= read -r app_name; do
    APP_NAME="$app_name" yq 'select(.kind == "Application" and .metadata.name == strenv(APP_NAME))' \
      "$output/applications.yaml" > "$output/app-$root_name-$app_name.yaml"
  done < <(yq -N -r 'select(.kind == "Application") | .metadata.name' "$output/applications.yaml")
done < <(find clusters -path '*/argocd/kustomization.yaml' | sort)
while IFS= read -r file; do
  while IFS= read -r path; do
    test -f "$path/kustomization.yaml"
    kubectl kustomize "$path" > "$output/child.yaml"
  done < <(yq -r '.spec.sources[] | select(.path != null) | .path' "$file")
  source_path=$(yq -r '.spec.source.path // ""' "$file")
  if [[ -n "$source_path" ]]; then
    test -f "$source_path/kustomization.yaml"
    kubectl kustomize "$source_path" > "$output/child.yaml"
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
  if [[ "$chart" == cert-manager && "$version" == v1.20.4 ]]; then
    scripts/check-cert-manager.sh "$output/helm.yaml" "$output"
  fi
  echo "Rendered $file"
done < <(find "$output" -name 'app-*.yaml' | sort)
echo "Validated $count Kustomizations and all Application source paths and Helm values."
