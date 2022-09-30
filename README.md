# Temporalite

[![Go Reference](https://pkg.go.dev/badge/github.com/temporalio/temporalite.svg)](https://pkg.go.dev/github.com/temporalio/temporalite)
[![ci](https://github.com/temporalio/temporalite/actions/workflows/ci.yml/badge.svg)](https://github.com/temporalio/temporalite/actions/workflows/ci.yml)
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

Download and extract the [latest release](https://github.com/temporalio/temporalite/releases/latest) from [GitHub releases](https://github.com/temporalio/temporalite/releases).

Start Temporal server:

```bash
temporalite start --namespace default
```

At this point you should have a server running on `localhost:7233` and a web interface at <http://localhost:8233>.

### Use CLI

Use [Temporal's command line tool](https://docs.temporal.io/tctl) `tctl` to interact with the local Temporalite server.

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

By default the web UI is started with Temporalite. The UI can be disabled via a runtime flag:

```bash
temporalite start --headless
```

To build without static UI assets, use the `headless` build tag when running `go build`.

### Dynamic Config

Some advanced uses require Temporal dynamic configuration values which are usually set via a dynamic configuration file inside the Temporal configuration file. Alternatively, dynamic configuration values can be set via `--dynamic-config-value KEY=JSON_VALUE`.

For example, to disable search attribute cache to make created search attributes available for use right away:

```bash
temporalite start --dynamic-config-value system.forceSearchAttributesCacheRefreshOnRead=true
```

## Development

To compile the source run:

```bash
go build -o dist/temporalite ./cmd/temporalite 
```

To run all tests:

```bash
go test ./...
```

## Known Issues

- When consuming Temporalite as a library in go mod, you may want to replace grpc-gateway with a fork to address URL escaping issue in UI. See <https://github.com/temporalio/temporalite/pull/118>
