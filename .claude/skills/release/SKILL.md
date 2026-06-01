---
name: release
description: Cut a release by merging develop into main and tagging it. Two-phase and gated — Phase 1 opens the develop→main PR and queues it to auto-merge as a merge commit once reviewed; Phase 2 tags main vX.Y.Z to fire the release draft. Never squashes develop→main.
---

# Release

## Goal

Promote `develop` to `main` and tag it for deployment, while keeping `main` and `develop`
history **shared**. The single rule that everything else protects:

> **`develop → main` is merged with a MERGE COMMIT, never squashed or rebased.**

Squashing severs shared history: `main` gets a brand-new commit that doesn't have develop's
tip as an ancestor, so the next `develop → main` PR shows *all* of develop's commits as
"missing from main" and reports phantom add/add conflicts. A merge commit keeps develop's tip
as an ancestor of main, so future releases show only genuinely new commits.

(For contrast, `feat/* → develop` *is* squash-merged — see the `open-pr` skill — and that's
fine, because feature branches are short-lived and deleted. The rule only bites between two
long-lived branches.)

### Repo prerequisites (one-time, already configured)

These make merge-commit releases possible — verify if a release ever misbehaves:

- `main` branch protection has **"Require linear history" OFF** (linear history forbids merge
  commits). Check: `gh api repos/<owner>/<repo>/branches/main/protection --jq '.required_linear_history.enabled'` → must be `false`.
- Repo setting **"Allow auto-merge" ON** and **"Allow merge commits" ON**. Check:
  `gh api repos/<owner>/<repo> --jq '{allow_auto_merge, allow_merge_commit}'` → both `true`.
- `main` requires **1 approving review** from someone other than the author (e.g. a maintainer),
  so a release PR always needs a teammate's approval before it merges.

## Detect the phase

Always start with `git fetch origin --tags --prune`, then pick the phase:

- **Nothing to do** — `git rev-list --count origin/main..origin/develop` is `0` and `origin/main`
  is already tagged. Say so and stop.
- **Phase 2 (tag)** — `origin/main` carries the release merge but has no tag yet (main is ahead
  of the latest `v*` tag), or a `develop→main` PR was just merged. Tag it.
- **Between phases** — an OPEN PR with base `main`, head `develop` already exists. If it's queued
  to auto-merge it just needs its review; point the user there (Phase 1 step 7). Do not open another.
- **Phase 1 (open PR)** — develop is ahead of main and no open release PR exists. Open it.

## Phase 1 — open the release PR

1. `git fetch origin --tags --prune`. Confirm there's something to ship:
   `git log origin/main..origin/develop --no-merges --oneline`. If empty → stop: "Nothing to release."
2. Bail if a release PR is already open:
   `gh pr list --base main --head develop --state open`. If one exists, point to it and stop.
3. **Pre-flight: is `develop` green?** `main` has no required status checks, so a broken `develop`
   could ship. Check the latest CI run on develop:
   `gh run list --branch develop -L 1 --json conclusion,status,workflowName`.
   If the latest run isn't `success` (failing, or still in progress), **warn and ask** before
   continuing — don't release a red branch.
4. **Suggest the version** (semver, `vMAJOR.MINOR.PATCH`):
   - Last tag: `git describe --tags --abbrev=0 origin/main 2>/dev/null` (repo's first real
     release baseline is `v0.0.1`).
   - Scan `git log <last-tag>..origin/develop --no-merges`:
     - any `BREAKING CHANGE` or `type!:` → bump **major**
     - else any `feat:` → bump **minor**
     - else (`fix:`/`chore:`/`refactor:`/…) → bump **patch**
   - If no tag exists at all → propose `v0.1.0`.
   - **Show the proposed version and ask the user to confirm or override.** Never tag from a guess.
5. **Draft the PR body** — terse, grouped, ~150 words max (same density rules as `open-pr`):
   ```md
   ## Release vX.Y.Z

   <one-line summary of the release>

   ### Highlights
   - <feat bullet>
   - <feat bullet>

   ### Fixes
   - <fix bullet, only if notable>
   ```
   Derive bullets from the actual commits since the last tag. Don't invent; don't list migration
   files, go.sum bumps, or mock regenerations.
6. Open the PR:
   ```bash
   gh pr create --base main --head develop --title "chore: release vX.Y.Z" --body "$(cat <<'EOF'
   <body>
   EOF
   )"
   ```
7. **Queue auto-merge as a merge commit — do NOT leave the merge button to chance.** This is the
   whole point of the skill: GitHub merges the PR *as a merge commit* the moment its required
   review lands, so nobody can accidentally squash it.
   ```bash
   gh pr merge <num> --auto --merge
   ```
   Then tell the user, plainly:
   > Release PR #<num> is queued to **auto-merge as a merge commit** once approved.
   > It needs **one approval from a maintainer** (not the author). Ping a reviewer.
   > After it merges, re-run `/release` to tag and ship.

   If `--auto --merge` is rejected with "merge method merge commits are not allowed", the repo
   prerequisites above have drifted (linear history got re-enabled, or merge commits disabled) —
   fix those first, then re-queue. Never fall back to a squash or rebase merge.

## Phase 2 — tag the release

1. `git fetch origin --tags --prune`. Identify the merged release (most recently merged
   `develop→main` PR, or `origin/main` ahead of the latest `v*` tag).
2. **Safety check — did the merge keep shared history?**
   ```bash
   git merge-base --is-ancestor origin/develop origin/main && echo OK || echo BROKEN
   ```
   - If **BROKEN**, the release PR was squashed or rebased. **STOP and warn loudly:** shared
     history is severed and the next release will hit phantom conflicts. Offer the repair before
     tagging — open a fresh `develop→main` PR and merge it as a **merge commit**
     (`gh pr merge <num> --merge`); its content diff is empty but it re-links ancestry. Only tag
     once the check prints `OK`.
3. **Determine the version** — read it back from the merged release PR title
   (`chore: release vX.Y.Z`). Confirm with the user.
4. **Tag-existence guard.** Make sure the tag isn't already used:
   `git ls-remote --tags origin "refs/tags/vX.Y.Z"` (and `git tag -l vX.Y.Z` locally).
   If it already exists, **stop** — pick a new version or investigate; never move an existing tag.
5. ⚠️ **Warn before tagging — this is the production gate.** Pushing the tag triggers
   `.github/workflows/release.yml`, which creates a **draft** GitHub Release. Publishing that
   draft triggers the prod deploy. State that plainly and get explicit confirmation.
6. Tag main's merge commit and push the tag:
   ```bash
   git fetch origin
   git tag -a vX.Y.Z origin/main -m "Release vX.Y.Z"
   git push origin vX.Y.Z
   ```
7. Tell the user: a **draft** release now exists (Releases page / Actions → release workflow).
   Edit the notes and **publish** it to deploy. The skill stops here — it never publishes.

## Rules

- **NEVER** squash or rebase `develop → main`. Merge commit only.
- **NEVER** push a version tag without explicit confirmation — it leads to a prod deploy.
- Version tags are `vMAJOR.MINOR.PATCH` — leading `v`, no suffixes, no omissions.
- Derive release notes from real commits since the last tag; never invent them.
- Keep output terse and skimmable, matching the `open-pr` skill.
