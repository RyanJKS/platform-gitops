# Platform GitOps Manifests

Contains k8s cluster add-ons, Argo CD resources and application deployments

## Getting started

Use Python 3.12. Create a virtual environment and install the development tools:

```sh
python -m venv .venv
source .venv/bin/activate
python -m pip install -r requirements-dev.txt
pre-commit install
python src/main.py
```

On Windows PowerShell, activate with `.venv\Scripts\Activate.ps1` instead.

## Checks

```sh
pre-commit run --all-files
python -m compileall -q src
```

CI runs these checks on pushes to `main` and pull requests. Compilation checks syntax; add behavior tests as the application grows.

## Layout

- `src/`: application code.
- `infrastructure/`: infrastructure configuration when needed.
- `docs/architecture.md`: design, dependencies, and operational decisions.
- `.github/`: CI, dependency updates, ownership, and contribution templates.

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidance and [SECURITY.md](SECURITY.md) for vulnerability reporting.

## Project setup

- Confirm GitHub code owners have write access.
- Configure branch protection or rulesets and require the `Checks` CI job.
- Enable private vulnerability reporting where available.
- Select a license before distributing the code.
- Replace the starter architecture notes and document any runtime configuration.

## Releases

Record changes in [CHANGELOG.md](CHANGELOG.md). Use semantic version tags such as `v0.1.0` when ready to release. Tags do not trigger publishing or deployment.
