# Architecture

## Purpose

Contains k8s cluster add-ons, Argo CD resources and application deployments

## Current design

The initial application is a Python 3.12 script in `src/main.py`. Running it prints `hello world` and exits. It has no runtime dependencies, network services, database, or deployment configuration.

## Repository boundaries

- `src/` contains application code.
- `infrastructure/` is reserved for infrastructure definitions.
- `.github/workflows/` contains CI automation.
- `docs/` contains architecture and operational documentation.

## Development checks

Pre-commit checks formatting, lint rules, YAML, and common repository mistakes. GitHub Actions runs the same checks and compiles Python source. These checks do not replace behavior tests.

## Design decisions to record

As the application grows, document:

- Main components and how data flows between them.
- External services, runtime dependencies, and ownership.
- Configuration, secret storage, and trust boundaries.
- Deployment, rollback, monitoring, and recovery procedures.
- Important tradeoffs and links to architecture decision records.
