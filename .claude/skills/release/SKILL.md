---
name: release
description: Cut a release in one command — open the develop→main PR and queue it to auto-merge as a merge commit. A maintainer's approval then triggers merge → tag → draft release automatically. Never squash develop→main.
---

# Release

Promote `develop` to `main` and ship it in **one command**. You open the release PR; once a
maintainer approves, the rest is automatic:

```
/release → PR opened + auto-merge queued → (maintainer approves)
        → GitHub merges as a MERGE COMMIT → release.yml tags vX.Y.Z + auto-drafts the notes
        → you publish the draft → deploy.yml ships the tag to Railway prod
```

## The one rule

> **`develop → main` must merge as a MERGE COMMIT — never squash or rebase.**

Squashing gives `main` a commit that doesn't have develop's tip as an ancestor, so the next
release PR shows every develop commit and hits phantom conflicts. A merge commit keeps history
shared. (Squashing `feat/* → develop` is fine — those branches are short-lived; see `open-pr`.)
The `--auto --merge` in step 6 enforces this.

## How it's wired

- **Auto-merge** merges the PR as a merge commit the instant its required review lands.
- **`release.yml`** runs on the merged PR: reads the version from the title
  (`chore: release vX.Y.Z`), tags main's merge commit, **auto-generates a categorized changelog**
  from the commits, and creates the release **as a draft**. You don't hand-write release notes.
- **`deploy.yml`** runs when you **publish** that draft: it deploys the released tag to Railway
  production. Publishing is the single go-live gate.

**Prerequisites** (already configured — check these if a release misbehaves):

- `main` protection: "Require linear history" **off**; requires **1 non-author review**.
- Repo: "Allow auto-merge" and "Allow merge commits" **on**.
- `release.yml` triggers on `pull_request: [closed]` → `main`; `deploy.yml` on `release: [published]`.
- Railway's native auto-deploy is **off** (deploy goes through `deploy.yml`); `RAILWAY_TOKEN` secret set.

## Steps

1. `git fetch origin --tags --prune`. If `git log origin/main..origin/develop --no-merges --oneline`
   is empty → stop: "Nothing to release."
2. If a `develop→main` PR is already open (`gh pr list --base main --head develop --state open`),
   report its status and stop — don't open another.
3. **Pre-flight.** `main` has no required checks, so confirm develop is green:
   `gh run list --branch develop -L 1 --json conclusion,status`. If the latest run isn't
   `success`, warn and ask before continuing.
4. **Version.** Propose the next semver and have the user confirm — never guess silently:
   - Last tag: `git tag -l 'v*.*.*' --sort=-v:refname | head -1` (baseline `v0.0.1`).
   - Bump from the commits since that tag: `BREAKING CHANGE`/`type!:` → **major**; else `feat:` →
     **minor**; else → **patch**.
5. **Open the PR.** Title MUST be exactly `chore: release vX.Y.Z` — `release.yml` parses the
   version from it. Body is the PR description for the reviewer (the published release notes are
   auto-generated): terse grouped highlights (~150 words, same density as `open-pr`) from real
   commits; skip migration files, go.sum bumps, mock regens.
   ```bash
   gh pr create --base main --head develop --title "chore: release vX.Y.Z" --body "$(cat <<'EOF'
   ## Release vX.Y.Z

   <one-line summary>

   ### Highlights
   - <feat bullet>

   ### Fixes
   - <fix bullet, if notable>
   EOF
   )"
   ```
6. **Queue auto-merge as a merge commit:** `gh pr merge <num> --auto --merge`. If rejected with
   "merge method merge commits are not allowed", a prerequisite drifted — fix it and re-queue.
   Never fall back to squash or rebase.
7. **Report and stop** — don't tag, merge, or publish; those are the automation's and the user's
   jobs:
   > Release PR #<num> (`vX.Y.Z`) is queued to auto-merge as a merge commit. It needs **one
   > maintainer approval** (not the author). On approval it merges, then `release.yml` tags it and
   > drafts the notes — review the draft and **publish** it; that deploys to Railway prod.
