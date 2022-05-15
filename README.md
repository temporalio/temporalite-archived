# Temporalite

[![Go Reference](https://pkg.go.dev/badge/github.com/DataDog/temporalite.svg)](https://pkg.go.dev/github.com/DataDog/temporalite)
[![ci](https://github.com/DataDog/temporalite/actions/workflows/ci.yml/badge.svg)](https://github.com/DataDog/temporalite/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/DataDog/temporalite/branch/main/graph/badge.svg)](https://codecov.io/gh/DataDog/temporalite)

> ⚠️ This project is experimental and not suitable for production use. ⚠️

Temporalite is a distribution of [Temporal](https://github.com/temporalio/temporal) that runs as a single process with zero runtime dependencies.

Persistence to disk and an in-memory mode are both supported via SQLite.

_Check out this video for a brief introduction and demo:_ [youtu.be/Hz7ZZzafBoE](https://youtu.be/Hz7ZZzafBoE?t=284) [16:13] -- demo starts at [11:28](https://youtu.be/Hz7ZZzafBoE?t=688)

## Why

The primary goal of Temporalite is to make it simple and fast to run Temporal locally or in testing environments.

Features that align with this goal:
- Easy setup and teardown
- Fast startup time
- Minimal resource overhead: no dependencies on a container runtime or database server
- Support for Windows, Linux, and macOS
- Ships with a web interface

## Getting Started

### Download and Start Temporal Server Locally

Build from source using [go install](https://golang.org/ref/mod#go-install):

> Note: Go 1.18 or greater is currently required.

> Note: On Windows, first, make sure that you have Go and GCC installed:
> * Go: https://go.dev/doc/install
> * GCC: https://jmeubank.github.io/tdm-gcc/ (pick an install for both 32 & 64 bit)

```bash
go install github.com/DataDog/temporalite/cmd/temporalite@latest
```

Start Temporal server:

```bash
temporalite start --namespace default
```

At this point you should have a server running on `localhost:7233` and a web interface at http://localhost:8233.

### Use CLI

Use [Temporal's command line tool](https://docs.temporal.io/docs/system-tools/tctl) `tctl` to interact with the local Temporalite server.

```bash
tctl namespace list
tctl workflow list
```

## Configuration

Use the help flag to see all available options:

```bash
temporalite start -h
```

### Namespace Registration

Namespaces can be pre-registered at startup so they're available to use right away:

```bash
temporalite start --namespace foo --namespace bar
```

Registering namespaces the old-fashioned way via `tctl --namespace foo namespace register` works too!

### Persistence Modes

#### File on Disk

By default `temporalite` persists state to a file in the [current user's config directory](https://pkg.go.dev/os#UserConfigDir). This path may be overridden:

```bash
temporalite start -f my_test.db
```

#### Ephemeral

An in-memory mode is also available. Note that all data will be lost on each restart.

```bash
temporalite start --ephemeral
```

### Web UI

The `temporalite` binary can be compiled to omit static assets for installations that will never use the UI:

```bash
go install -tags headless github.com/DataDog/temporalite/cmd/temporalite@latest
```

The UI can also be disabled via a runtime flag:

```bash
temporalite start --headless
```
