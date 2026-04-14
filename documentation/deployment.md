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

## GCP Cloud Build Setup

All build logic lives in the `Dockerfile` at the root of the repository. Cloud Build uses a minimal `cloudbuild.yaml` to orchestrate: build the image, push it to Artifact Registry, and deploy to Cloud Run.

### `cloudbuild.yaml`

```yaml
substitutions:
  _REGION: europe-west1            # override per trigger if needed
  _SERVICE: fifawcp-dev            # set to fifawcp-dev or fifawcp-prod in each trigger
  _REGISTRY: europe-west1-docker.pkg.dev  # Artifact Registry host

steps:
  # Build the Docker image using the Dockerfile.
  # Tagged with the commit SHA for full traceability.
  - name: gcr.io/cloud-builders/docker
    args:
      - build
      - -t
      - $_REGISTRY/$PROJECT_ID/$_SERVICE:$COMMIT_SHA
      - .

  # Push the image to Artifact Registry.
  - name: gcr.io/cloud-builders/docker
    args:
      - push
      - $_REGISTRY/$PROJECT_ID/$_SERVICE:$COMMIT_SHA

  # Deploy the new image to Cloud Run.
  - name: gcr.io/google.com/cloudsdktool/cloud-sdk
    args:
      - gcloud
      - run
      - deploy
      - $_SERVICE
      - --image=$_REGISTRY/$PROJECT_ID/$_SERVICE:$COMMIT_SHA
      - --region=$_REGION
      - --platform=managed
      - --quiet

images:
  - $_REGISTRY/$PROJECT_ID/$_SERVICE:$COMMIT_SHA
```

### Creating the Triggers

1. Go to **GCP Console → Cloud Build → Triggers → Create trigger**
2. Connect your GitHub repository
3. Create two triggers:

**Dev trigger**

- Name: `fifawcp-deploy-dev`
- Event: Push to branch — regex `^develop$`
- Configuration: `cloudbuild.yaml`
- Substitutions: `_SERVICE=fifawcp-dev`

**Prod trigger**

- Name: `fifawcp-deploy-prod`
- Event: Push to branch — regex `^main$`
- Configuration: `cloudbuild.yaml`
- Substitutions: `_SERVICE=fifawcp-prod`

> Adjust `_REGION` and `_REGISTRY` to match your GCP project location and Artifact Registry repository.

---

## Secrets & Environment Variables

Runtime secrets (DB credentials, JWT secret, etc.) should be stored in **GCP Secret Manager** and injected into Cloud Run at deploy time — not stored in GitHub.

In the GCP console: **Cloud Run → Service → Edit → Variables & Secrets**

Or via CLI:

```bash
gcloud run deploy fifawcp-prod \
  --update-secrets=DB_URL=db-url:latest,JWT_SECRET=jwt-secret:latest
```

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
