# ADR-020: Docker container lifecycle and invocation

Status: Accepted
Date: 2026-08-14

## Context

The isolation seam has one implementation, `StubEngine`. It does not run code. Every event the engine records today describes an execution that never happened.

This ADR adds the first real implementation of the seam. It covers how jhansi talks to Docker, what a containers's lifetime is, and how a container is invoked. It does not cover timeouts, output caps, or what happens when Docker is missing. Those decisions depend on the invocation shaoe settled here and follow in ADR-021 and ADR-022.

## Decision

### Transport: the Docker daemon socket

The driver speaks HTTP to the Docker daemon over its unix socket at `/var/run/docker.sock`. It uses stdlib `net/http` with a custom `DialContext` that dials the socket. It does not shell out to the `docker` CLI.

Both approaches are stdlib-only, so the CLI is not simpler here. Two reasons decide it.

First, cancellation. With `exec.CommandContext`, cancelling the context kills the `docker` client process. The container keeps running. A per-exec timeout would then need a second shell-out to `docker kill` and reconciliation of what actually dies. Over the socket the driver holds the container ID and calls `POST /containers/{id}/kill` directly.

Second, failure diagnosis. Over the socket, Docker not installed arrives as `ENOENT` on the socket path, a stopped daemon arrives as `ECONNREFUSED`, and a missing image arries as a 404 from the create call. Via the CLI these are all English text on stderr, and the wording varies across Docker versions.

There is a third reason that is smaller but real. `docker run jhansi` with the socket mounted is jhansi's own distribution shape. Shelling out would mean shipping the `docker` binary inside that image.

Accepted cost: when TTY is off, `GET /containers/{id}/logs` returns a multiplexed stream with an 8-byte frame header per chunk. Splitting stdout from stderr needs roughly forty lines of demuliplexing.

### Lifecycle: one container per exec

Each `Exec` creates a fresh container, runs it, reads its result, and removes it. Nothing survives the call except the working directory, which lives on jhansi's disk and is bind-mounted in.

A container that outlives a request would be a resource with its own lifetime. Reaping it would drag the sandbox TTL and reaper work forward. This ADR keeps the container lifetime inside the request.

### The daemon must be local

The sandbox working directory is bind-mounted into the container. Bind mounts are resolved by the daemon, not by the client. A daemon on another host looks for the path on its own filesystem, does not find it, and silently creates an empty directory. The container then runs with none of the sandbox's files, and nothing errors.

Pointing jhansi at a remote daemon is therefore not a transport change. It is a different filesystem design, using `PUT /containers/{id}/archive` to copy files in and out. This ADR requires a local daemon and states the reason so the constraint is not mistaken for an oversight.

### Invocation

- **Mount and working directory.** The sandbox directory from ADR-019 is bind mounted at `/workspace`. The container's working directory is also `/workspace`. One path, not two concepts.
- **User.** The containers runs as the UID and GID of the jhansi process. This is a correctness decison before it is a hardening one. A container running as root writes root-owned files into the bind mount, and sandbox deletion can then fail to remove the directory.
- **Network.** The container uses Docker's default bridge network. Restricting egress is a separate decision and gets its own ADR. Setting `NetworkMode: none` here would break the first dependency install and hide a policy decision inside a driver.
- **Removal.** The driver removes the container explicitly after reading the exit code and draining the logs. `AutoRemove` is not used, because it races the log read and can destroy output the engine is about to read.

### Image

`image` is an optional field on sandbox create. When absent, the server's configured default is used. The default is set by an operator flag on `jhansi server` and ships as `python:3.12-slim`.

The engine attaches no meaning to the default. It is a convenience for callers who do not care, not a statement about what jhansi runs. A caller running Go or Java passes an image and never sees it.

A `language` field was considered and rejected. Docker takes an image, not a language, so a language field would require jhansi to own a mapping table from language names to images. That table would need a release every time someone wants a version or a language it lacks. The image is the primitive and a language shortcut can be built on it later. The reverse is not true.

## Consequences

- The isolation seam has a working implementation and executions become real.
- Sandboxes are language-neutral in practice, not only in principla.
- The zero-configuration path runs Python without the engine being a Python runtime.
- Installing dependencies is an ordinary exec, not engine behaviour. The engine never inspects a workspace to decide what to install.
- jhansi now has an external runtime dependency, and the operator must have Docker running on the same host.
- A wrong or unpulled image reaches the caller as an error from the create call, which is what makes the optional field safe for automated callers.

## Testing

- The driver builds the expected create request from a sandbox directory, an image, and a command.
- Exit code, stdout, and stderr are read back from a real container and returned as an `ExecResult`.
- Multiplexed log output is demultiplexed into separate stdout and stderr streams.
- Files written by the container into `/workspace` appear in the sandbox directory on the host and are owned by the jhansi user.
- The container is removed after the result is read.

## Deferred

- **Per-exec timeout and output cap** — ADR-021. `TimedOut` exists on `ExecResult` and nothing can set it yet.
- **Docker unavailable** — ADR-022. The three-way taxonomy of not installed, daemon down, and image missing.
- **Image pull policy** — whether jhansi ever pulls an image, or requires the operator to have pulled it.
- **Restricting which images a caller may run** — this needs an authenticated caller, and there is no auth seam yet.
- **Network policy** — default-deny egress and allowlisting.
- **Containers that outlive a request** — the trigger is a user needing state that is not a file to survive between execs. Files already survive, and preinstalled dependencies are answered by base images.
- **Binding a container port to a host port** — only meaningful once a container outlives a request.
