#!/usr/bin/env bash
set -euo pipefail

# An environment name alone does not enforce approval. Fail closed if its
# required-reviewer policy is missing, or the API cannot verify the policy.
: "${GITHUB_REPOSITORY:?GitHub repository is required}"
required_reviewers=$(gh api \
  "repos/$GITHUB_REPOSITORY/environments/aks-envoy" \
  --jq 'any(.protection_rules[]?; .type == "required_reviewers" and (.reviewers | length > 0))')
if [[ "$required_reviewers" != true ]]; then
  echo "::error::Configure required reviewers on the aks-envoy GitHub environment before bootstrap can apply."
  exit 1
fi
