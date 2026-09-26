# Contributing

1. Create a branch from `main` and keep changes focused.
2. Run `scripts/validate.sh` with the tool versions listed in `.github/workflows/validate.yaml`.
   This needs network access for pinned charts and Kubernetes schemas.
3. Review rendered resources for every affected cluster. Validation does not contact a Kubernetes API
   and cannot confirm runtime readiness, cloud identities, DNS, or secret access.
4. Update documentation and the changelog when behavior or operations change.
5. Open a pull request describing the problem, resulting behavior, and validation.

Run the same strict documentation build as CI from the repository root:

```sh
docker run --rm -v "$PWD:/content" -w /content spotify/techdocs:v1.2.6 \
  mkdocs build --strict --site-dir /tmp/site
```

An existing MkDocs environment with `mkdocs-techdocs-core` can instead run
`mkdocs build --strict --site-dir /tmp/platform-gitops-docs`.
Keep `mkdocs.yml`, `docs/`, and the `dir:.` TechDocs annotation in `catalog-info.yaml` aligned.
Do not commit generated documentation, secrets, local kubeconfig files, or infrastructure state.

Require the `Manifests` and `TechDocs` jobs in branch protection. Chart version updates require
review of upstream compatibility and migration notes. Dependabot maintains GitHub Actions only.

## Configuration placement

Read [configuration boundaries](docs/configuration-boundaries.md) before adding defaults or cluster features.
Shared catalog changes must be portable across their consumers. Put permissions, namespace policy,
quotas, networking, TLS, identities, storage, and operational sizing under the target cluster.
Identical policy values are not a reason to centralize independent cluster decisions.
Add the feature's validation and operational documentation in the same change.
