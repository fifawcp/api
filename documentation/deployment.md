# Deployment Guide

## Branch Strategy

```md
feature/* ──► develop ──► main ──► tag v*.*.*
                │            │          │
          GCP trigger   GCP trigger  release.yml
          (dev deploy)  (prod deploy) (draft release)
```

- **`feature/*`** — short-lived branches for individual changes. Always branch off `develop`.
- **`develop`** — integration branch. Every merge automatically deploys to the dev environment via a GCP Cloud Build trigger.
- **`main`** — production-ready code. Every merge automatically deploys to production via a GCP Cloud Build trigger.
- **Tags (`v*.*.*`)** — create a tag when you want to cut a release. This creates a draft GitHub Release for you to write the changelog and publish.

---

## CI/CD Overview

Deploys are handled by **GCP Cloud Build triggers**, not GitHub Actions.
GitHub Actions is responsible only for quality gates (lint, build, tests) and release drafting.

```md
.github/workflows/
  ci.yml        # lint + build + unit tests — runs on every push and PR
  release.yml   # creates a draft GitHub Release when a version tag is pushed
```

GCP Cloud Build triggers (configured in the GCP console):

| Trigger name          | Branch    | Deploys to                 |
| --------------------- | --------- | -------------------------- |
| `fifawcp-deploy-dev`  | `develop` | `fifawcp-dev` (Cloud Run)  |
| `fifawcp-deploy-prod` | `main`    | `fifawcp-prod` (Cloud Run) |

---

## Day-to-Day Development

```md
# 1. Branch off develop
git checkout develop
git pull origin develop
git checkout -b feature/my-feature

# 2. Work, commit, push
git push origin feature/my-feature

# 3. Open a PR → develop
#    GitHub Actions runs: lint + build + unit tests

# 4. Merge PR → develop
#    GitHub Actions runs ci.yml again
#    GCP trigger fires → deploys to dev automatically
```

---

## Releasing to Production

```md
# 1. Open a PR: develop → main (CI must pass)
# 2. Merge → main
# 3. GCP trigger fires → deploys to production automatically

# 4. Tag the release
#   git checkout main
#   git pull origin main
#   git tag v1.2.3
#   git push origin v1.2.3

# release.yml creates a draft GitHub Release with a raw commit log

# 5. Go to GitHub → Releases → find the draft
# 6. Clean up the commit log into readable release notes
# 7. Click "Publish release"
```

> Production deploys on merge to `main`. The tag and release are for **versioning and changelog** only — they do not trigger an additional deploy.

---

## Full Flow Summary

| Step | Action | Handled by |
| ---- | ------ | ---------- |
| Push feature branch | CI runs (lint, build, test) | GitHub Actions |
| PR → `develop` | CI runs on PR | GitHub Actions |
| Merge → `develop` | Deploy to dev | GCP Cloud Build |
| PR → `main` | CI runs on PR | GitHub Actions |
| Merge → `main` | Deploy to production | GCP Cloud Build |
| `git tag v1.2.3` | Draft release created | GitHub Actions |
| Publish release | Changelog published | You (manual) |

---

## Release Notes Template

Use this template when writing release notes for GitHub Releases:

```md
# Release vX.Y.Z

Brief summary of the release.

## Highlights

- Highlight 1
- Highlight 2
- Highlight 3

## What's changed

### Features
- Added ...

### Improvements
- Improved ...

### Bug fixes
- Fixed ...

### Testing
- Added/updated ...

### Developer experience
- Added/updated ...

### Internal
- Refactored ...
- Updated dependencies ...

## Breaking changes

None.

## Migration notes

- Run ...
- Update ...
- Verify ...

## Deployment notes

- Required environment variables:
  - `VAR_NAME`
- Infra notes:
  - ...
- Operational notes:
  - ...

## Contributors

- @username
- @username

## Full diff

[PREV...vX.Y.Z](https://github.com/OWNER/REPO/compare/PREV...vX.Y.Z)
```
