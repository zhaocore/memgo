---
name: generate-zhao-style-agents-md-skill
description: Generate or update a repo-specific `AGENTS.md` that improves coding-agent stability and output quality through concrete architecture guardrails, testing rules, documentation workflow, product invariants, verification instructions, and focused user Q&A when key repo decisions are missing. Use when the user wants an `AGENTS.md` tailored to an existing codebase or a new repo plan rather than generic agent guidance.
---

# Generate zhao-style AGENTS.md skill

Generate an `AGENTS.md` that turns engineering preferences into concrete operating rules for coding agents in one specific repository or planned repository.

## Goals

- Keep changes aligned with the repository's real structure, product boundaries, and testing setup.
- Reduce drift between requirements, code, docs, and tests.
- Make verification, documentation, and approval expectations explicit.
- Work for both existing repos and new repos that are still being defined.

## Core workflow contract

This skill defines the non-negotiable workflow that generated `AGENTS.md` files must encode. The output must operationalize these principles directly, not merely gesture at them.

Unless the user explicitly asks for a deliberate deviation, the generated `AGENTS.md` must require:

- adherence to clearly defined backend abstractions
- adherence to clearly defined frontend abstractions
- unit test coverage written together with every backend code change
- frontend unit test coverage written together with every frontend code change
- Playwright coverage written together with every frontend code change
- user requirements maintained in `docs/REQUIREMENTS.md`, structured with up to three levels of hierarchy
- every task tracked chronologically as checklist items in `docs/PLAN_md/PLAN_yymmdd.md`, and checked off when done
- every change compared against current requirements, with ambiguous conflicts flagged to the user
- every change verified by a full unit test run, with ambiguous failures flagged to the user
- user feedback that reveals a behavior mismatch reflected back into both requirements and tests

Do not weaken these into optional suggestions such as "if needed", "if already present", "if the repo wants it", or "for non-trivial changes". These rules apply to every change and every task unless the user explicitly requests a deliberate deviation. When a concern is absent today, such as a repo with no frontend yet, preserve the principle as a forward-looking rule instead of dropping it. When required workflow files or commands are missing, instruct the agent to create or standardize them instead of silently omitting the rule.

## Determine repo mode first

Choose the mode before writing:

- Existing repo: inspect the repository and derive guidance from code, docs, tests, and current structure.
- New repo or mostly-empty repo: derive guidance from the available plan, requirements, scaffolding, repo docs, and focused user answers.

For a new repo, it is acceptable for `AGENTS.md` to describe intended structure rather than observed structure, but clearly keep those statements aligned with what the user has confirmed.

## For new repos, help the user make choices

When the repo is new or mostly empty, do more than gather facts. Help the user choose a clean starting point that will keep agent behavior stable as the codebase grows.

Use these default recommendations unless the user or repo context points elsewhere:

- Prefer the fewest runtime boundaries that fit the product. Do not split into extra services or workers without a real operational reason.
- Split out a worker only when the product has long-running, retryable, or separately scalable jobs.
- Define clear placement rules early so business logic stays out of transport, UI glue, and integration wrappers, and so backend and frontend abstractions stay explicit.
- Use `docs/REQUIREMENTS.md` as the canonical requirements document.
- Use chronological checklist plan files at `docs/PLAN_md/PLAN_yymmdd.md`.
- Require unit tests from day one for backend and shared business logic.
- If frontend code exists or is planned, require frontend unit tests and Playwright coverage from day one.
- Put external systems behind explicit seams or ports so tests can mock or fake them.
- Mark generated, vendor, cache, build, and sensitive config paths as off-limits for product logic.
- If frontend code exists, keep reusable state, formatting, and business rules out of DOM glue.

## Inspect the repo first

Before drafting or updating `AGENTS.md`, inspect the repository and derive the guidance from what is actually present.

Read the most relevant sources first:

- Repo root layout and major directories
- App entrypoints, workers, scripts, and shared modules
- Test directories and actual test commands
- Repo overview, onboarding, or architecture docs
- `docs/REQUIREMENTS.md` if present
- Any architecture, logging, security, config, or pre-commit docs
- Any existing `AGENTS.md`, contributor guide, or similar local instructions

Do not assume any of the following without confirming them in the repo:

- A specific `backend/` or `frontend/` directory name exists
- A specific unit-test or Playwright command already exists
- The current dated plan file already exists
- One shared abstraction scheme fits every repo

For a new repo, do not invent a detailed file tree unless the user has already chosen one. Ask for the intended boundaries first, then write the smallest concrete guidance that fits.

## Derive repo facts before writing

Capture only facts that are true for this repo:

- Where product logic belongs
- Which directories or files are entrypoints versus reusable modules
- Which paths should never hold product logic
- Which abstraction boundaries are already in use
- Which product or domain invariants must stay true
- Which tests must be updated when specific areas change
- Which commands should be run for verification
- Which docs are canonical for requirements or architecture
- Which changes require explicit approval notes
- Which security, logging, or config constraints matter

If docs and code disagree, prefer the current code and passing tests first, then instruct the agent to update stale docs in the same change.

For a new repo, if plans, docs, and user answers disagree, surface the conflict and resolve it before baking it into `AGENTS.md`.

## Output requirements

Write an `AGENTS.md` that is concrete, repo-aware, and actionable.

Every generated `AGENTS.md` must clearly encode the full workflow contract above, even when the exact section titles differ.

Prefer:

- Real paths, filenames, modules, and commands
- Repo-specific responsibility boundaries
- Clearly marked intended boundaries when the repo is new
- Short checklists for architecture-impacting edits
- Explicit test expectations tied to touched areas
- Domain guardrails when the product has important invariants
- Practical warnings about legacy files, drift, or known traps

Avoid:

- Generic advice that could apply to any repository
- Invented directories, packages, tools, or workflows
- Omitting the core workflow rules just because the repo is small or new
- Hardcoding fake paths or fake commands for Playwright, plans, or frontend/backend code
- Replacing a repo's real structure with a shared template
- Presenting guesses as if they were confirmed repo facts

## Minimum section coverage for K-style output

The final `AGENTS.md` may merge titles when that reads better, but it must clearly cover the following topics.

### Start Here

Use to tell the agent what to read and update before making changes. Do not scope the core workflow rules to only larger or "non-trivial" work.

### Repository Map

List key entrypoints, modules, and directories with one-line responsibilities. For a new repo, this can describe the intended top-level structure instead of current files.

### Architecture Rules

State where transport, orchestration, domain logic, integrations, config, backend logic, and frontend logic should live in this repo. For a new repo, define the intended placement rules clearly enough that early implementation stays consistent.

### Frontend Rules

If the repo has frontend code, mention real entrypoints, reusable logic boundaries, and the rule that every frontend change carries unit tests and Playwright coverage. If the repo has no frontend yet, add a short forward rule saying any future frontend work must define a clear boundary and carry those tests.

### Domain Guardrails

Include when the product has fixed flows, invariants, lifecycle rules, or data contracts that should not drift.

### Testing Policy

State that every backend code change requires unit test coverage, every frontend code change requires frontend unit tests plus Playwright coverage, and every change requires a full unit test run. Include the real commands used in this repo, or explicitly call out the missing command gap that must be resolved.

### Documentation Rules

State that `docs/REQUIREMENTS.md` is the canonical requirements document, that it is a living document, that it must be created if missing, and that requirements should be organized with up to three levels of hierarchy.

### Planning Rules

State that work should be tracked chronologically in checklist form in `docs/PLAN_md/PLAN_yymmdd.md`, that the relevant dated file should be created if missing, and that items should be checked off when completed.

### Consistency Checks

State that every change must be compared against the current requirements, and that ambiguous conflicts must be flagged to the user instead of guessed through.

### Feedback Loop

State that when user testing or feedback reveals a mismatch, both the written requirements and the relevant tests should be refined so the expected behavior becomes more precise over time.

### Security and Config Hygiene

Include when the repo has secrets, config files, logging constraints, or sensitive integrations that agents must handle carefully.

### Files and Paths to Avoid

Call out generated, vendor, environment-managed, fixture, cache, or sensitive paths that must not become homes for product logic.

### Change Approval

List the kinds of architecture-impacting edits that require explicit notes in requirements docs, review notes, or commit messages.

## Required generation heuristics

When writing or updating `AGENTS.md`:

- Preserve strong repo-specific guidance that already exists.
- Tighten vague rules into concrete ones using the repo's actual files and commands.
- When the repo is new, distinguish intended structure from observed structure.
- Use `docs/REQUIREMENTS.md` as the canonical requirements document and mention the up-to-three-level hierarchy rule.
- If `docs/REQUIREMENTS.md` does not exist yet, instruct the agent to create it rather than silently weakening the rule.
- Treat requirements as a living document that must be updated when behavior changes or gets clarified.
- Require unit tests for every backend code change.
- Require frontend unit tests and Playwright coverage for every frontend code change. If the repo has no frontend yet, say so explicitly and state that the rule activates when frontend code is introduced.
- Require a full unit test run for every change, and tell the agent to flag ambiguous failures to the user.
- Require every change to be checked against the current requirements, and tell the agent to flag ambiguous conflicts to the user.
- Require chronological checklist tracking in `docs/PLAN_md/PLAN_yymmdd.md`. If the file does not exist yet, instruct the agent to create or update it rather than silently dropping the rule.
- Do not scope the mandatory workflow rules to "non-trivial" work, "significant" work, or any other fuzzy threshold.
- When user feedback exposes a mismatch, instruct the agent to update both the relevant tests and `docs/REQUIREMENTS.md`.
- When frontend Playwright coverage is required, prefer idempotent or self-cleaning flows.
- Mention MCP availability only when it is relevant to the repo's workflow.
- If the repo has large legacy entrypoints, instruct agents to preserve behavior and extract downward instead of rewriting broadly.
- If the repo depends on external systems, instruct tests to mock or fake them unless the repo clearly expects integration tests.

## Basic user Q&A

Ask questions only when the answer cannot be discovered from the repo or when the repo is new enough that the structure is still undecided.

Keep the first round short. Ask only the minimum set needed to avoid encoding bad assumptions.

Good starter questions:

- What are the main product boundaries in this repo, such as frontend, backend, worker, CLI, jobs, or scripts, and what paths should own them? Recommend the fewest boundaries that fit the product, and recommend a separate worker only for long-running or retryable jobs.
- What paths or modules are intended to own each boundary? If the user has no preference yet, propose a small top-level structure instead of a large framework-shaped tree.
- What are the real unit-test and Playwright commands, or if they do not exist yet, what commands should this repo standardize on? For repos with frontend work, recommend both frontend unit coverage and Playwright from the start.
- How should `docs/REQUIREMENTS.md` be structured so requirements stay organized with up to three levels of hierarchy?
- Should each task stream get its own dated checklist file under `docs/PLAN_md/PLAN_yymmdd.md`, or should the repo append to a daily file when work spans multiple tasks? Recommend the simplest chronological convention that still keeps tasks readable.
- Are there generated, vendor, fixture, cache, or sensitive paths that agents should never use for product logic? If the repo is still empty, suggest likely forbidden paths based on the chosen stack.
- Which changes count as architecture-impacting and therefore require explicit notes in requirements docs, review notes, or commit messages? Help the user name the risky categories early.
- What product behaviors, lifecycle rules, or invariants must not drift? Push for concrete flows and acceptance-critical states rather than abstract goals.

Prefer asking these in priority order:

1. Product boundaries and ownership
2. Required test layers
3. Canonical requirements and planning docs
4. Sensitive or forbidden paths
5. Architecture-impacting change categories
6. Domain invariants

If the user does not know all answers yet, propose a minimal default and label it as a recommendation rather than an observed fact.

When the user is undecided, present at most 2-3 reasonable options, explain the tradeoff in one sentence each, and recommend one.

## Suggested authoring flow

1. Determine whether this is an existing repo or a new repo.
2. Inspect the repo layout, major modules, docs, and test setup if code already exists.
3. Ask focused user questions if key facts are missing or the repo is still being designed.
4. Identify the real or intended architectural boundaries and areas of drift.
5. For a new repo, recommend simple defaults and explain tradeoffs when the user is undecided.
6. Identify repo-specific domain invariants and risky change categories.
7. Draft the mandatory core workflow sections plus any extra repo-specific sections that add value for this repo.
8. Replace generic statements with concrete paths, commands, and examples.
9. Run the final workflow-alignment self-check before returning the draft.
10. Keep the document short enough to stay readable, but specific enough to guide coding agents reliably.

## Final workflow-alignment self-check

Before returning a generated or updated `AGENTS.md`, verify that it:

- requires clearly defined backend abstractions to be followed at all times
- requires clearly defined frontend abstractions to be followed at all times, or explicitly states the forward-looking rule for future frontend code
- requires unit test coverage with every backend code change
- requires frontend unit test coverage with every frontend code change
- requires Playwright coverage with every frontend code change
- requires `docs/REQUIREMENTS.md` to be the canonical requirements document, organized with up to three levels of hierarchy, and created if missing
- requires every task to be tracked chronologically in checklist form in `docs/PLAN_md/PLAN_yymmdd.md`, with the relevant file created if missing
- requires every change to be compared against current requirements, with ambiguous conflicts flagged to the user
- requires every change to be verified by a full unit test run, with ambiguous failures flagged to the user
- requires user feedback that reveals a mismatch to be reflected back into both requirements and tests
- does not weaken mandatory rules with phrases like "if needed", "if already present", "if the repo wants it", or "for non-trivial changes"

## Success criteria

The generated `AGENTS.md` should help a coding agent answer these questions without guessing:

- What files should I read first?
- Where should this kind of logic live?
- What paths or files should I avoid?
- What requirements doc is canonical?
- Which tests must I update and run?
- What product behavior must not drift?
- Which changes need explicit approval notes?
- Which parts of this guidance are observed repo facts versus intended structure?
- If this repo is new, which defaults were recommended and why?
