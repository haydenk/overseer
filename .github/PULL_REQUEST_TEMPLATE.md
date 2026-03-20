## Description

<!-- What does this PR do? Summarise the change and link the related issue. -->

Closes #

---

## Type of change

- [ ] Bug fix &nbsp;&nbsp;(`bugfix/<name>`)
- [ ] New feature &nbsp;&nbsp;(`feature/<name>`)
- [ ] Hotfix &nbsp;&nbsp;(`hotfix/<name>`)
- [ ] Release prep &nbsp;&nbsp;(`release/<version>`)
- [ ] Documentation / chore

## Changes

<!-- Bullet-point summary of what changed. -->

-

## Testing

<!-- How was this verified? What should reviewers specifically check? -->

- [ ] `go test ./...` passes locally
- [ ] Tested against a real Procfile (`overseer start`)
- [ ] Signal handling verified (Ctrl-C, SIGTERM) if relevant

## Procfile / .env changes

- [ ] This PR affects Procfile parsing or `.env` handling — example Procfile updated in README if applicable

---

## Checklist

- [ ] Branch follows gitflow naming (`feature/`, `bugfix/`, `hotfix/`, `release/`)
- [ ] `gofmt` run and no lint warnings introduced (`mise run check`)
- [ ] `CHANGELOG.md` updated (for features and bug fixes)
- [ ] PR title is descriptive and suitable for a changelog entry
