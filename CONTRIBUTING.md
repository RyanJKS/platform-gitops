# Contributing

Follow the setup steps in [README.md](README.md), including `pre-commit install`.

1. Create a branch from `main` with a descriptive name.
2. Keep changes focused and add tests for new behavior when applicable.
3. Run `pre-commit run --all-files` and `python -m compileall -q src`.
4. Update documentation and the changelog when behavior or setup changes.
5. Open a pull request describing the problem, changes, and validation.

Use clear commit messages. Never commit secrets, local environment files, or infrastructure state. Document new configuration with safe example values.

To update hook versions, run `pre-commit autoupdate`, run all checks, and review the resulting changes before committing.
