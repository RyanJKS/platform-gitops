# Changelog

## Unreleased

### Added

- An end-to-end Mermaid diagram for the `webapp-mysql-pv` sandbox, covering the
  selected webapp and MySQL resources and persistent storage.
- Optional ARC controller and runner scale-set catalog defaults, namespace bases, validated examples, and onboarding guidance.

- Shared platform defaults, namespaces, gateway resources, and workload bases with minimal cluster overlays.
- Azure dev shared and dedicated cluster roots, plus local kind bootstrap.
- Reserved Azure production and AWS development cluster directories.
- Bootstrap, onboarding, promotion, recovery, and architecture guides.
- Manifest rendering, schema validation, and strict TechDocs checks in CI.

### Changed

- Named the sandbox MySQL Service `mysql` to match its connection hostname and
  init container check. Updated the diagram and documented Service DNS naming.
- Split the webapp/MySQL sandbox into nested `mysql/` and `webapp/` Kustomizations,
  keeping a root entry point for combined applies without bases or overlays and
  a root-owned `webapp-mysql` Namespace.
  The webapp child includes its Deployment and Service. Fixed the MySQL
  Deployment API version and replica count, and the PVC access modes field.
- Renamed cloud cluster directories from `spoke-atlas/` to `atlas/`, using the domain
  name directly. Updated Application paths, validation fixtures, and documentation;
  external root owners must update their source paths to match.

- Grouped environments beneath subscription/account aliases, using `DEV-JKS` for the current Azure subscription.

- Renamed cluster configuration from `live/` to `clusters/` and removed redundant inner cluster directories.

- Cluster overlays now own platform permissions, namespace security labels, quotas, Gateway listeners, and controller sizing.
- Documented developer boundaries for shared installation defaults and future cluster features.

### Removed

- Python starter application, Python tooling, pre-commit configuration, and unused infrastructure scaffold.
