# Contributing

Thanks for your interest in Dune Awakening Shop!

## Workflow

- `main` is always releasable.
- Work on feature branches: `feat/<topic>`, `fix/<topic>`, `docs/<topic>`.
- Open a Pull Request into `main`. PRs are merged with `--no-ff` to keep history.
- Commits follow [Conventional Commits](https://www.conventionalcommits.org/):
  `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`, `ci:`.

## Before pushing

```bash
gofmt -s -w .
go vet ./...
go build ./...
go test ./...
```

CI (GitHub Actions) runs build, vet and tests on every push and PR.

## Versioning

[Semantic Versioning](https://semver.org/). Tags `vX.Y.Z` cut a release.

## Code style

- Standard Go conventions; keep functions small and testable.
- All SQL lives in `internal/store`.
- Wrap errors with context (`fmt.Errorf("...: %w", err)`).
- Never commit secrets. Local config is `config.yaml` (git-ignored).
