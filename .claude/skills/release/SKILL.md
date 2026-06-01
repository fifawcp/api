---
name: release
description: Cut a release by merging develop into main and tagging it. Two-phase and gated — Phase 1 opens the develop→main PR with a merge-commit reminder; Phase 2 (after you merge) tags main vX.Y.Z to fire the release draft. Never squashes develop→main.
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

## Detect the phase

Always start with `git fetch origin --tags --prune`, then pick the phase:

- **Nothing to do** — `git rev-list --count origin/main..origin/develop` is `0` and `origin/main`
  is already tagged. Say so and stop.
- **Phase 2 (tag)** — `origin/main` carries the release merge but has no tag yet (main is ahead
  of the latest `v*` tag), or a `develop→main` PR was just merged. Tag it.
- **Between phases** — an OPEN PR with base `main`, head `develop` already exists. Point the user
  to it and the merge instruction (Phase 1 step 6); do not open another.
- **Phase 1 (open PR)** — develop is ahead of main and no open release PR exists. Open it.

## Phase 1 — open the release PR

1. `git fetch origin --tags --prune`. Confirm there's something to ship:
   `git log origin/main..origin/develop --no-merges --oneline`. If empty → stop: "Nothing to release."
2. Bail if a release PR is already open:
   `gh pr list --base main --head develop --state open`. If one exists, point to it and stop.
3. **Suggest the version** (semver, `vMAJOR.MINOR.PATCH`):
   - Last tag: `git describe --tags --abbrev=0 origin/main 2>/dev/null` (repo's first real
     release baseline is `v0.0.1`).
   - Scan `git log <last-tag>..origin/develop --no-merges`:
     - any `BREAKING CHANGE` or `type!:` → bump **major**
     - else any `feat:` → bump **minor**
     - else (`fix:`/`chore:`/`refactor:`/…) → bump **patch**
   - If no tag exists at all → propose `v0.1.0`.
   - **Show the proposed version and ask the user to confirm or override.** Never tag from a guess.
4. **Draft the PR body** — terse, grouped, ~150 words max (same density rules as `open-pr`):
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
5. Open the PR:
   ```bash
   gh pr create --base main --head develop --title "chore: release vX.Y.Z" --body "$(cat <<'EOF'
   <body>
   EOF
   )"
   ```
6. **Print this prominently — it is the whole point of the skill:**

   > ⚠️ **Merge this PR with "Create a merge commit" — NOT "Squash and merge".**
   > Squashing severs shared history and breaks every future release.
   > Foolproof from the CLI: `gh pr merge <num> --merge`
   > Then re-run `/release` to tag and ship.

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
4. ⚠️ **Warn before tagging — this is the production gate.** Pushing the tag triggers
   `.github/workflows/release.yml`, which creates a **draft** GitHub Release. Publishing that
   draft triggers the prod deploy. State that plainly and get explicit confirmation.
5. Tag main's merge commit and push the tag:
   ```bash
   git fetch origin
   git tag -a vX.Y.Z origin/main -m "Release vX.Y.Z"
   git push origin vX.Y.Z
   ```
6. Tell the user: a **draft** release now exists (Releases page / Actions → release workflow).
   Edit the notes and **publish** it to deploy. The skill stops here — it never publishes.

## Rules

- **NEVER** squash or rebase `develop → main`. Merge commit only.
- **NEVER** push a version tag without explicit confirmation — it leads to a prod deploy.
- Version tags are `vMAJOR.MINOR.PATCH` — leading `v`, no suffixes, no omissions.
- Derive release notes from real commits since the last tag; never invent them.
- Keep output terse and skimmable, matching the `open-pr` skill.
