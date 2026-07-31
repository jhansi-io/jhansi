# ADR-016: Bootstrap — cmd/jhansi

Status: Accepted
Date: 2026-07-30

## Context

Everything through ADR-015 is reachable only from the test suite. There is no
binary. The engine needs a `main` that constructs the real registry, sink,
isolation engine and handler, and serves them.

Four things must be decided to write that `main`, and each has a cheap answer
now and an expensive one later:

1. How the operator supplies configuration
2. Where the event sink file lands on disk.
3. Which `http.Server` timeouts are set — the stdlib defaults are none.
4. Whether the binary takes a subcommand.

## Decision

**1. Configuration is command-line flags.** stdlib `flag`, no environment
variables, no config file. Flags are self-documenting via `-h`. Environment
variables are what a containerised user asks for; that request has not arrived.

**2. A `-data-dir` flag, not an events-file flag.** Default `./jhansi-data`,
created by `main` if absent, mode `0700`. The sink writes to
`<data-dir>/events.jsonl`.

A directory is chosen over a file path because a second consumer of the same
root is already designed: the Tier 1 run log dir at
`<data-dir>/runs/<run_id>/stdout`. A file flag would force an unrelated second
path flag later with no shared root. Mode `0700` because that tree will
eventually hold captured process output, which can contain secrets the runtime
saw.

This is not the Tier 5 data-directory work. Layout conventions, backup/restore
and `jhansi doctor` remain Tier 5; this decision only establishes that a root
exists.

**3. `ReadHeaderTimeout` and `IdleTimeout` are set. `WriteTimeout` is not.**

`ReadHeaderTimeout` closes the slowloris hole left by the stdlib defaults.
`IdleTimeout` is free. `WriteTimeout` is a wall-clock cap on the entire
response, and the exec route is synchronous — a real container exec will run
for seconds to minutes. Any value chosen today would have no rationale, because
the per-exec timeout it must exceed does not exist yet.

**4. The binary dispatches on a subcommand: `jhansi server`.** Exactly one
subcommand ships. Flags are parsed with `flag.NewFlagSet("server", flag.ExitonError)` over `os.Args[2:]`, because package-level `flag.Parse` stops
at the first non-flag argument. A missing or unknown subcommand prints usage to
stderr and exits 2.

The subcommand is decided now rather than later because `jhansi server` is the
documented install experience, and it is the string users script against.
Adding a verb after release  means supporting both `jhansi -addr :8080` and
`jhansi server -addr :8080` indefinitely, or breaking people.

## Consequences

- The engine is runnable: `jhansi server` starts an HTTP server on `-addr`
  (default `:8080`) with the real `Registry`, `FileSink` and `StubEngine`.
- `main` owns construction and wiring. Nothing else gains a dependency.
- A failure to create the data directory or open the sink is fatal at startup — 
  the process exits non-zero rather than serving without an evidence spine.
- Adding a second subcommand later is additive: another `case` and another
  `FlagSet`.
- `WriteTimeout` remaining unset is deliberative, recorded gap. It is coupled to
  the Tier 1 per-exec timeout and must be revisited in the same change, not
  before.
- Construction (`newServer`) is separated from listening (`runServer`) so the
  wired handler and the real `FileSink` are testable without binding a port.
  Graceful shutdown will build on the same split.

## Deferred

- **Per-exec timeout, and with it `WriteTimeout`** — Tier 1.
- **CLI shape** — subcommand taxonomy, output conventions, exit codes, `help`,
  `version`. This ADR decides only that `os.Args[1]` is a verb.
- **Graceful shutdown, `jhansi doctor`, Prometheus metrics, config file, auth** — 
  out of scope.
- **Environment-variable configuration** — on demand.
- **Failure surface** — non-zero exit → FAILED, `TimedOut` → TIMED_OUT, infra
  error → sandbox ERROR, typed 409 on a busy sandbox — ADR-017.
