# Contributing

For larger changes, open an issue first so the approach is agreed on before
you invest the work.

Contributions are submitted as GitHub pull requests. External contributors
should fork the repository and create a branch in their fork. Repository
collaborators may create a branch directly in this repository. Keep changes
focused and explain their user-visible, compatibility, and security impact.

## Setup

Tool versions are managed with [mise](https://mise.jdx.dev):

```sh
mise install
```

## Acceptance Requirements

Contributions must be formatted, pass lint checks, and pass the full test
suite. Changes to exported behavior must update the relevant documentation.
New functionality must include automated tests, and bug fixes must include a
regression test. If an automated test is impractical, explain why and describe
the manual verification in the pull request.

Run the required checks before opening a pull request:

```sh
make fmt
make lint
make test
```

Run `make help` for all targets. CI also enforces module tidiness, security
scans, cross-platform tests, and an 80 percent coverage threshold.

## Compatibility

Testastic is a public Go library. Treat removed exported identifiers, changed
signatures, and incompatible behavior changes as breaking changes. Call out
compatibility impact explicitly in the pull request.

## Commits

Commit messages must follow [Conventional Commits](https://www.conventionalcommits.org/)
in the form `type(scope): description`. `feat`, `fix`, and `perf` commits
trigger a release. Other commit types do not.

## Releases

Releases are automated by yeet. Do not bump versions, edit `CHANGELOG.md`, or
create tags manually.
