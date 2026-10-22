# Implement on another device with Qwen

## Scope and readiness

This checkout contains planning only for Vikunja task 114. No application changes, production database operations or deployment have run. Six work packages contain 24 subtasks; each is assigned to Qwen with the implementer profile. No local Qwen executable is needed to create or transfer these files.

Read `spec.md`, `plan.md`, `research.md`, `tasks.md` and the package prompts. The package graph is WP01 → WP02/WP03 → WP04 → WP05 → WP06. Execute sequentially on this mission's single integration branch; final reviewed changes can be proposed to `master` in a PR.

## Transfer

Planning branch: `plan/postgres-dashboard`. The planning commit(s) are local; nothing has been pushed. Transfer the Git bundle supplied with the planning result, or push this branch yourself and fetch it on the implementation device. The bundle includes repository history and the planning branch; it does not include credentials or ignored local environment files.

On the other device, clone the supplied bundle into a working directory:

```bash
git clone -b plan/postgres-dashboard /path/to/postgres-dashboard-planning.bundle trago-server-postgres
cd trago-server-postgres
```

The bundle is the clone's initial origin. Set the repository's intended upstream remote before publishing any implementation changes. No upstream URL is prescribed here.

## Prepare the implementation device

Install Qwen CLI and Spec Kitty using your normal installation method. Use Spec Kitty 3.2.7 or a compatible newer version. Verify binaries and commands on that device:

```bash
qwen --version
spec-kitty --version
spec-kitty agent config add qwen
spec-kitty agent config status
spec-kitty mission-type list
```

The project tracks Codex and Qwen as available agents. Host-global Qwen command files and runtime caches are installed on each device and are not transferred in Git. If the CLI reports missing skills or commands, run the supported setup/doctor repair there. If mission context reports `Unknown mission type 'software-dev'`, repair/activate the installed software development mission type through the charter/setup workflow; do not fabricate phase status or mark packages implemented. This machine encountered that error for `spec-kitty plan`, although task validation succeeds.

Prepare Go/CGO, Node/Yarn, repository code generation and a disposable PostgreSQL 17 instance as required by the Makefile and plan. Existing tests use SQLite; WP01 defines the new PostgreSQL test configuration. Do not point tests at the live CNPG database.

## Qwen starting prompt

Start Qwen in the repository root and give it this instruction:

> Implement Spec Kitty mission `postgres-dashboard-01M5JRH2` from its committed spec, plan and tasks. First use `/spec-kitty.analyze` for this mission and resolve blocking findings; persist `analysis-report.md` before claiming any implementation work. Then use the canonical Spec Kitty workflow with agent identity `qwen`, one package at a time, starting at WP01. Preserve SQLite and PostgreSQL 17 support, including legacy bytea columns, and record actual test evidence. Follow the six package prompts. Do not deploy, publish an image, modify the external automation repository or close Vikunja task 114 without subsequent instructions.

After the required analysis gate passes, the runtime entry point is:

```bash
spec-kitty next --agent qwen --mission postgres-dashboard-01M5JRH2
```

Follow the returned action and workspace instructions. Do not start by marking packages done or manually creating worktrees. `spec-kitty next` is an agent-facing control loop, not a command that launches Qwen by itself.

## What was verified here

`spec-kitty agent mission finalize-tasks --mission postgres-dashboard-01M5JRH2 --target-branch plan/postgres-dashboard --validate-only --json` returned `validation_passed` for all six packages, with no ownership or requirement extraction warnings. The real finalization result is recorded in the planning session output and commits. Backend tests have not run because no application code was implemented. The analysis gate intentionally remains for the implementation device.
