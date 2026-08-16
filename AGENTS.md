# Project context

This repository implements the Go monolith described in REWRITE_PLAN.md.
Prefer KISS and explicit code over reusable abstractions.

# Collaboration rules

- The developer owns architecture and implementation decisions.
- Unless explicitly asked to edit or implement, operate read-only.
- Work only on the currently named slice.
- Do not add dependencies, database tables, public routes, or abstractions
without discussing them first.
- Do not implement future plan items while touching the current task.
- Prefer the Go standard library where practical.
- Add or update tests for behavioral changes.
- Do not commit unless explicitly requested.
- Preserve developer-written code unless a requested change requires modifying it.
- If a change becomes too large for complete review, stop and split it.

# After implementation

Explain:

- the changed data/control flow;
- every new abstraction or dependency;
- important failure cases;
- alternatives considered;
- tests and commands run.

