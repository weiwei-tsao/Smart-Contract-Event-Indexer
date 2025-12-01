# Documentation Rules

## Core Principle: Less is More

Only document what provides **ongoing value**. Avoid temporary records and redundant summaries.

---

## Document Types & Rules

### ✅ MAINTAIN (Update existing, don't create new)

| Document | Path | When to Update |
|----------|------|----------------|
| Progress | `docs/PROGRESS.md` | Phase completion, metrics change |
| Quick Reference | `docs/QUICK_REFERENCE.md` | Config or API changes |
| Phase TODO | `docs/PHASE_X_TODO.md` | Current phase only, delete when done |

### ✅ KEEP (Core documentation)

| Document | Path | Purpose |
|----------|------|---------|
| PRD | `docs/smart_contract_event_indexer_prd.md` | Requirements (rarely update) |
| Architecture | `docs/smart_contract_event_indexer_architecture.md` | Design (rarely update) |
| Plan | `docs/smart_contract_event_indexer_plan.md` | Roadmap (rarely update) |
| API Docs | `docs/api/*.md` | API reference |
| Runbooks | `docs/deployment/*.md` | Operations |
| ADRs | `docs/architecture/decisions/*.md` | Major decisions only |

### ❌ DO NOT CREATE

- `*_SUMMARY.md` files
- `*_COMPLETE.md` files
- `*_FINDINGS.md` files
- Debug session logs
- PR description files
- Duplicate phase records

---

## Naming Conventions

```
docs/
├── PROGRESS.md              # Single progress tracker
├── QUICK_REFERENCE.md       # Developer quick ref
├── PHASE_X_TODO.md          # Current phase only (delete after)
├── DOCUMENTATION_RULES.md   # This file
├── api/
│   └── {service}_api.md     # e.g., query_service_api.md
├── deployment/
│   └── {service}_runbook.md # e.g., indexer_runbook.md
├── development/
│   └── GIT_WORKFLOW.md      # Git workflow only
└── architecture/
    ├── decisions/
    │   └── XXX-{topic}.md   # e.g., 001-why-microservices.md
    └── diagrams/
        └── {name}.md        # Architecture diagrams
```

---

## Update Rules

### When completing a task:
1. Update `PROGRESS.md` checkboxes
2. Update metrics if changed
3. **Do NOT** create completion summary files

### When completing a phase:
1. Update `PROGRESS.md` phase table
2. Delete `PHASE_X_TODO.md` for completed phase
3. Create `PHASE_Y_TODO.md` for next phase (if needed)
4. **Do NOT** create `PHASE_X_COMPLETE.md`

### When making architecture decisions:
1. Update existing ADR if topic exists
2. Create new ADR only for **major** decisions
3. **Do NOT** create decision summaries

---

## Validation Checklist

Before creating any new doc, ask:
- [ ] Does this info already exist elsewhere? → Update existing
- [ ] Will this be useful in 30 days? → If no, don't create
- [ ] Is this a summary of something? → Don't create
- [ ] Can this go in PROGRESS.md? → Put it there

---

## Anti-Patterns (Avoid)

```
❌ DOCUMENT_REVIEW_FINDINGS.md     → Put findings in PROGRESS.md
❌ PHASE_2_IMPLEMENTATION_SUMMARY.md → Update PROGRESS.md instead
❌ FINAL_RESOLUTION_SUMMARY.md     → No meta-documentation
❌ 2025-01-20-debug-session.md     → Don't log debug sessions
❌ pr_phase4.md                    → PRs are in git, not docs
```

---

## File Count Target

| Category | Max Files |
|----------|-----------|
| Root docs/ | 7 (includes 3 core docs + rules) |
| api/ | 3 |
| deployment/ | 3 |
| development/ | 1 (GIT_WORKFLOW only) |
| architecture/ | 3 (decisions + diagrams) |
| **Total** | ~15 |

If docs/ exceeds 15 files, consolidate.

---

**Rule Version**: 1.0
**Last Updated**: 2025-12-01
