---
name: release
description: Cut a release in one command — open the develop→main PR and queue it to auto-merge as a merge commit. Once a maintainer approves, GitHub merges it and the release workflow tags + drafts the release automatically. Never squashes develop→main.
---

# Release

## Goal

Promote `develop` to `main` and ship it — in a **single command**. You open the release PR;
everything after a maintainer approves it happens automatically:

```
/release  ──>  PR opened + auto-merge queued
                     │  (a maintainer approves)
                     ▼
          GitHub auto-merges it as a MERGE COMMIT
                     │  (release.yml fires on the merge)
                     ▼
          tag vX.Y.Z pushed  +  DRAFT GitHub Release created
                     │  (you edit + publish the draft)
                     ▼
                prod deploy
```

The single rule everything protects:

> **`develop → main` is merged with a MERGE COMMIT, never squashed or rebased.**

Squashing severs shared history — `main` gets a new commit that doesn't have develop's tip as an
ancestor, so the next release PR shows *all* of develop's commits and reports phantom conflicts.
A merge commit keeps history shared. (Feature branches `feat/* → develop` *are* squashed — that's
fine, they're short-lived; see the `open-pr` skill. The rule only bites between two long-lived
branches.)

## How the automation is wired

- **Auto-merge** (GitHub) merges the release PR the instant its required review lands — as a
  merge commit, so it can't be accidentally squashed.
- **`.github/workflows/release.yml`** runs on the merged `develop→main` PR: it reads the version
  from the PR title (`chore: release vX.Y.Z`), tags main's merge commit, and creates a **draft**
  GitHub Release. (The same workflow also handles a manual `vX.Y.Z` tag push for emergencies.)
  You publish the draft to deploy.

So this skill does **one thing**: open the PR with the right title and queue auto-merge.

### Repo prerequisites (one-time, already configured)

Verify these if a release ever misbehaves:

- `main` protection: **"Require linear history" OFF** —
  `gh api repos/<owner>/<repo>/branches/main/protection --jq '.required_linear_history.enabled'` → `false`.
- Repo: **"Allow auto-merge" ON** and **"Allow merge commits" ON** —
  `gh api repos/<owner>/<repo> --jq '{allow_auto_merge, allow_merge_commit}'` → both `true`.
- `main` requires **1 approving review** from a non-author (a maintainer).
- `release.yml` triggers on `pull_request: [closed]` to `main` (the auto path).

## The command

1. `git fetch origin --tags --prune`. Confirm there's something to ship:
   `git log origin/main..origin/develop --no-merges --oneline`. If empty → stop: "Nothing to release."
2. **Already in flight?** If a `develop→main` PR is open
   (`gh pr list --base main --head develop --state open`), don't open another — report its status
   (approved? merged yet?) and stop.
3. **Pre-flight — is `develop` green?** `main` has no required status checks, so a red develop
   could ship. Check `gh run list --branch develop -L 1 --json conclusion,status,workflowName`.
   If the latest run isn't `success`, **warn and ask** before continuing.
4. **Suggest the version** (semver, `vMAJOR.MINOR.PATCH`):
   - Last tag: `git tag -l 'v*.*.*' --sort=-v:refname | head -1` (baseline `v0.0.1`).
   - Scan `git log <last-tag>..origin/develop --no-merges`:
     - any `BREAKING CHANGE` / `type!:` → **major**; else any `feat:` → **minor**; else → **patch**.
   - **Show the proposed version and ask the user to confirm or override.** Never guess silently.
   - The chosen `vX.Y.Z` MUST go in the PR title verbatim — `release.yml` parses it from there.
5. **Draft the PR body** — terse, grouped, ~150 words max (same density as `open-pr`):
   ```md
   ## Release vX.Y.Z

   <one-line summary>

   ### Highlights
   - <feat bullet>

   ### Fixes
   - <fix bullet, if notable>
   ```
   Derive bullets from real commits since the last tag. Don't invent; skip migration files,
   go.sum bumps, mock regens.
6. **Open the PR** — the title carries the version, which is how the workflow learns it:
   ```bash
   gh pr create --base main --head develop --title "chore: release vX.Y.Z" --body "$(cat <<'EOF'
   <body>
   EOF
   )"
   ```
7. **Queue auto-merge as a merge commit** — never leave the merge button to chance:
   ```bash
   gh pr merge <num> --auto --merge
   ```
   If rejected with "merge method merge commits are not allowed", a prerequisite drifted (linear
   history re-enabled / merge commits disabled) — fix it, then re-queue. **Never** fall back to
   squash or rebase.
8. **Report and stop.** Tell the user, plainly:
   > Release PR #<num> (`vX.Y.Z`) is queued to **auto-merge as a merge commit**.
   > It needs **one maintainer approval** (not the author) — ping a reviewer.
   > On approval it merges, then `release.yml` tags `vX.Y.Z` and creates a **draft** release.
   > **Publish the draft** to deploy to production.

   The skill is done here — it does NOT tag, merge, or publish. Those are the automation's and
   the user's jobs.

## Rules

- **NEVER** squash or rebase `develop → main`. Merge commit only (enforced via `--auto --merge`).
- The PR title MUST be exactly `chore: release vX.Y.Z` — the workflow parses the version from it.
- **NEVER** push a tag by hand as part of this flow — `release.yml` owns tagging on the auto path.
  (A manual tag is only for emergencies and triggers the same workflow's manual path.)
- Confirm the version with the user; derive notes from real commits; keep output terse.
