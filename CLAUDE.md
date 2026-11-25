# AGENTS.md

This file provides guidance to agents working in this repository.

## Purpose and Precedence

- MUST means required.
- SHOULD means recommended unless there is a concrete reason to deviate.
- MAY means optional.
- Root `AGENTS.md` defines repository-wide defaults. If a deeper path later adds a more specific `AGENTS.md`, the deeper file SHOULD be treated as higher precedence for that subtree.

## Non-negotiables

1. Keep diffs minimal. Avoid unrelated refactors, broad renames, or formatting-only churn unless explicitly requested.
2. Leave verifiable evidence. Run targeted checks and report exact commands.
3. Respect generated code artifacts. Do not hand-edit generated code outputs; regenerate from source inputs.

## Agent Interaction Overrides (Repo-Local)

- For short non-code questions about definitions, acronyms, symbols, errors, or other potentially repository-specific terms, agents MUST search the active repository and read the closest authoritative local definition before answering, lead with the repository-specific meaning, and use general knowledge only when local evidence is absent or the user explicitly requests broader context.

## ExecPlans

When writing complex features or significant refactors, use an ExecPlan from design to implementation.

- Definition and format: `PLANS.md` at repository root.
- Requirement: keep the ExecPlan updated as a living document while implementation progresses.
- Scope: use this for multi-step work where losing context would risk correctness or incomplete validation.

## Code Style Guide

### Go and backend code

- Code SHOULD remain maintainable for future readers with basic concolic-testing familiarity, including readers who are not experts in the specific subsystem/feature.
- Follow existing package-local conventions first and keep style consistent with nearby files.
- Code SHOULD be self-documenting through clear naming and structure.
  - Example: when implementing a well-known algorithm, naming SHOULD be clear enough to make the approach recognizable; if naming alone may not make intent obvious, add a brief comment.
- Keep error handling actionable and contextual; avoid silently swallowing errors.
- Comments SHOULD explain non-obvious intent, constraints, invariants, concurrency guarantees, or important performance trade-offs, and SHOULD NOT restate what the code already makes clear.

### Tests and testdata

- Prefer extending existing test suites and fixtures over creating new scaffolding.
- Keep test changes minimal and deterministic; avoid broad golden/testdata churn unless required.
- When recording outputs, verify changed result files before reporting completion.

## Agent Output Contract

When delivering changes, report:

1. Files changed.
2. Risks: correctness, compatibility, performance.
3. Exact commands run for validation.
4. What was not verified locally.

For read-only analysis, report findings, supporting evidence, and verification limits; a code-validation profile is not required.

  ## Change Approval Workflow

  - Before editing, creating, or deleting any file, agents MUST first present the full proposed change set and wait for explicit user approval. Do not edit files in the same turn as the proposal.
  - The proposal MUST list, for each file: the path, the reason for the change, and the exact diff (or the new content for new files). Include any files outside this repository, such as `replace`d dependencies.   
  - Read-only investigation (reading files, grep, builds, running tests) does not need approval. Anything that modifies files, `go.mod`/`go.sum`, or generated outputs does.
  - After approval, apply only what was approved. If the implementation needs a change that wasn't in the proposal, stop and ask again.
  - A question such as "could you add X?" is a request for a proposal, not permission to edit.

