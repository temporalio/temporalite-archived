# Temporalite

> ⚠️ This project is experimental and not suitable for production use. ⚠️

Temporalite is a distribution of [Temporal](https://github.com/temporalio/temporal) that runs as a single process with zero runtime dependencies.

Persistence to disk and an in-memory mode are both supported via SQLite.

## Why

The primary goal of Temporalite is to make it simple and fast to run Temporal locally or in testing environments.

Features that align with this goal:
- Easy setup and teardown
- Fast startup time
- Minimal resource overhead: no dependencies on a container runtime or database server
- Support for Windows, Linux, and macOS

## Getting Started

### Download and Start Temporal Server Locally

Build from source using [go install](https://golang.org/ref/mod#go-install):

```bash
go install github.com/DataDog/temporalite@latest
```

Start Temporal server:

```bash
temporalite start
```

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
