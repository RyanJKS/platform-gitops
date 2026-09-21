---
name: documentation-maintainer
description: Maintain repository documentation alongside code changes, keeping MkDocs navigation, Markdown pages, and Backstage TechDocs metadata aligned. Use for documentation improvements and changes to behavior, setup, configuration, interfaces, architecture, or operations that affect readers.
---

# Documentation Maintainer

Keep documentation useful to someone who has not read the implementation or the
conversation. Make the smallest complete update supported by the actual code.

## Establish what changed

- Read repository instructions, the relevant diff, and the affected implementation.
  Preserve existing user edits and unrelated content.
- Read `mkdocs.yml`, `catalog-info.yaml`, the README, and the relevant documentation
  pages. Follow the current configuration if the repository has evolved beyond
  the starter layout.
- Identify the reader impact: behavior, commands, configuration, interfaces,
  limitations, architecture, or operations. Internal changes without reader impact
  need no forced documentation edit.
- Verify commands and claims against source, manifests, tests, and configuration.
  Do not document planned features as implemented or invent deployment procedures.

## Preserve the TechDocs contract

The starter uses this layout:

```text
catalog-info.yaml      Component metadata and TechDocs annotation
mkdocs.yml             Site configuration and navigation
docs/index.md          Documentation home page
docs/architecture.md   Current design and decisions
README.md              Repository overview and quick start
```

- Keep `metadata.annotations.backstage.io/techdocs-ref: dir:.` pointing to the
  directory containing `mkdocs.yml`, relative to the catalog descriptor. If these
  files move, update the reference and verify the new relationship.
- Keep `techdocs-core` in the MkDocs plugins list. Preserve existing plugins and
  extensions. Add a plugin, theme, or extension only after checking that the actual
  TechDocs build environment supplies it.
- Navigation targets are relative to `docs_dir` (normally `docs/`). Add useful new
  pages to navigation and update navigation when pages move or disappear. Keep
  the home page reachable.
- Use relative links between documentation pages and relative paths for images.
  Check filename case and heading anchors. A link such as `../README.md` from
  `docs/` usually points outside the published site; use an established source URL
  or put the needed material in the documentation site instead.
- Preserve catalog identity, owner, and type unless the requested change requires
  updating them. Do not replace project metadata with a personal account or sample
  organisation. Keep titles and descriptions consistent with the project.
- Keep generated `site/` output out of source control. Do not edit generated HTML
  or Backstage's published documentation storage as a documentation fix.

## Write for the reader

- Keep the README short: purpose, quick start, and a link to the documentation when
  a real URL is known. Put detailed guides in `docs/` and avoid maintaining the same
  procedure in two places.
- Make `docs/index.md` explain what the project does and where readers should go
  next. Organize navigation by reader tasks; add sections only when content merits
  them, rather than generating empty pages.
- For procedures, include prerequisites, exact commands, expected results, and
  relevant failure recovery. Use repository-supported tooling and label the shell
  or working directory when it matters. Never imply that an untested command ran.
- For configuration, explain defaults, required values, and operational effects.
  Use placeholders for secrets and identities, never actual credentials.
- Ground architecture in implemented components, data flow, dependencies, and
  tradeoffs. Distinguish current behavior from proposals. Add a diagram only when
  it clarifies the design and renders with the existing MkDocs configuration.
- Use descriptive headings, short paragraphs, fenced code blocks with language
  labels, and meaningful link text. Explain unfamiliar terms on first use.
  Prefer specific examples over marketing language or repeated summaries.

## Verify and report

1. Check changed pages against the implementation. Check navigation targets,
   relative links, images, and anchors affected by the edit. Search for stale names
   or paths when renaming a feature or page.
2. Use the repository's existing documentation check or build command. If MkDocs
   and `mkdocs-techdocs-core` are installed locally, a suitable fallback is:

   ```sh
   mkdocs build --strict --site-dir /tmp/documentation-maintainer-site
   ```

   Run from the directory containing `mkdocs.yml`; use a fresh temporary output
   directory for concurrent builds. Prefer pinned project tools or the configured
   TechDocs generator over installing arbitrary latest dependencies.
3. If the build toolchain is unavailable, perform the file and configuration checks
   you can and state the missing build check. Do not remove `techdocs-core` to make
   a generic MkDocs installation pass. Separate pre-existing build failures from
   failures introduced by the change.
4. Review the diff for accidental catalog changes, duplicated instructions, broken
   links, or generated files. Report what changed and which checks actually ran.
   A local MkDocs build does not prove Backstage publishing or access permissions.

For generator or deployment changes, consult the consuming Backstage configuration
and the [TechDocs guide](https://backstage.io/docs/features/techdocs/getting-started).
Documentation maintenance alone does not require changing that deployment.
