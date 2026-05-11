---
name: open-pr
description: Analyse the current branch's diff against develop, fill .github/pull_request_template.md, push the branch, and open the PR with `gh`. Output stays terse and skimmable.
---

# Open PR

## Goal

Open a PR whose body a reviewer can scan in under 30 seconds. Optimise for **signal density**, not coverage.

## Steps

1. **Gather context** in parallel:
   - `git log develop..HEAD --oneline` — list of commits on the branch
   - `git diff develop...HEAD --stat` — file-level overview
   - Current branch name (extract issue number if it matches `API-\d+`, e.g. `API-24` → `Closes #24`)
   - Read `.github/pull_request_template.md` to confirm section headers
   - Read enough of `git diff develop...HEAD` to understand intent — chunk if large; you do **not** need to read every hunk

2. **Draft the PR body** following the template exactly (three sections, no extras):

   ```md
   ## Related issue

   Closes #<number>

   ## What changed

   - <bullet>
   - <bullet>

   ## How to test

   - [ ] <step>
   - [ ] <step>
   ```

3. **Show the draft to the user** as a fenced markdown block. Ask: *"Open the PR with this body, or want me to adjust?"* Wait for confirmation.

4. **On approval**, push the branch if needed, then open the PR:
   - Check upstream: `git rev-parse --abbrev-ref --symbolic-full-name @{u} 2>/dev/null`
   - If no upstream is set, push first: `git push -u origin <branch>`
   - Then run:
     ```bash
     gh pr create --base develop --title "<title>" --body "$(cat <<'EOF'
     <body>
     EOF
     )"
     ```
   - Use the most recent commit subject as the title unless the user provided one.

5. **Return the PR URL.** Nothing else.

## Output rules — read carefully

These exist because past attempts produced 600-word PR bodies nobody read.

- **What changed: 3–6 bullets total.** One line each. Group by area only if there are clearly distinct areas. No nested bullets. No section sub-headers inside.
- **How to test: 3–6 checklist items.** Each one is a single concrete action: a curl command, a Swagger UI call, a DB query, or a log check. Cover the golden path. Skip exhaustive edge cases.
- **No preamble, no closing summary, no emoji.** No "This PR introduces…" framing. Start with the bullet.
- **Reference files only when it sharpens a bullet** — `internal/jobs/sync_match_results_job.go` is fine; deeply nested paths for obvious changes are noise.
- **Skip the obvious**: don't bullet migration file names, go.sum bumps, or mock regenerations.
- **Body cap: ~150 words.** If it's longer, you're listing instead of summarising — cut.

## Title rules

- Conventional Commits prefix (`feat`, `fix`, `chore`, `refactor`, `docs`).
- Under 70 chars.
- Use the dominant theme of the branch, not the latest commit if it's a small fixup.

## When to ask before drafting

- If the diff spans multiple unrelated themes, ask the user which is the PR scope before drafting — don't try to summarise everything.
- If there's no issue number derivable from the branch name (e.g. `API-24`), ask for one rather than guessing.
