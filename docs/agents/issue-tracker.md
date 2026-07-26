# Issue Tracker: Local Markdown

## Tracker Type

**Local markdown files** under `.scratch/<feature>/`

## Structure

```
.scratch/
├── <feature-slug>/
│   ├── issue.md        # Main issue description
│   ├── spec.md         # Spec/plan (from to-spec)
│   ├── tasks.md        # Task breakdown (from to-tickets)
│   └── notes/          # Optional scratch notes
```

## Workflow

1. **Create issue:** `mkdir -p .scratch/<feature-slug> && cat > .scratch/<feature-slug>/issue.md`
2. **Plan/spec:** Write `spec.md` in same directory
3. **Break down:** Write `tasks.md` with checkboxes
4. **Execute:** Update checkboxes in `tasks.md` as work progresses
5. **Archive:** Move `.scratch/<feature-slug>/` to `.archive/` on completion (optional)

## Tool Integration

Skills read/write this structure:
- `to-tickets` → creates `tasks.md`
- `to-spec` → creates `spec.md`
- `triage` → reads `issue.md`, updates labels in frontmatter
- `qa` → reads `spec.md` + `tasks.md` for verification

## Frontmatter (in issue.md)

```markdown
---
title: "Short title"
status: "open" | "in-progress" | "done" | "wontfix"
triage: "needs-triage" | "needs-info" | "ready-for-agent" | "ready-for-human" | "wontfix"
created: "YYYY-MM-DD"
updated: "YYYY-MM-DD"
labels: ["label1", "label2"]
---
```

## Conventions

- Feature slug: kebab-case, descriptive (`user-auth-jwt`, `api-rate-limit`)
- One feature per directory
- Keep `issue.md` as source of truth for "what"
- Keep `tasks.md` as source of truth for "how" and progress
- No GitHub CLI (`gh`) needed — pure file operations