# Domain Documentation Layout

## Layout: Single-Context

One `CONTEXT.md` at repo root + `docs/adr/` for Architecture Decision Records.

```
.
├── CONTEXT.md          # Project-wide context, conventions, key patterns
├── docs/
│   ├── adr/            # Architecture Decision Records (0001-title.md, ...)
│   └── agents/         # Agent skill configs (this directory)
```

## CONVENTIONS

### CONTEXT.md

- Project overview, tech stack, key architectural decisions
- Coding conventions specific to this repo
- Important file/module map (high-level only)
- Links to detailed ADRs in `docs/adr/`

### ADRs in `docs/adr/`

- Filename: `NNNN-short-title.md` (zero-padded 4-digit sequence)
- Format: Markdown with frontmatter
  ```markdown
  ---
  title: "Short Title"
  status: "accepted" | "proposed" | "deprecated" | "superseded"
  date: "YYYY-MM-DD"
  supersedes: ["0001-old-decision"]  # optional
  superseded-by: ["0003-new-decision"]  # optional
  ---
  ```

## Consumer Rules (for agents/skills reading domain docs)

1. **Read `CONTEXT.md` first** — always, for every task. It's the entry point.
2. **Follow ADR links** from `CONTEXT.md` when deeper context needed.
3. **Do not assume** `docs/agents/` contents are domain knowledge — they're skill configs only.
4. **Search `docs/adr/*.md`** for historical decisions when modifying related code.

## Maintenance

- Update `CONTEXT.md` when conventions change
- Add ADR for any architectural decision with trade-offs
- Keep `CONTEXT.md` concise (< 500 lines) — defer detail to ADRs