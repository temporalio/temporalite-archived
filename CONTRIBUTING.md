# Develop Temporalite
This doc is for contributors to Temporalite (hopefully that's you!)

[comment]: <> (TODO: CLA?)

[comment]: <> (**Note:** All contributors also need to fill out the [Temporal Contributor License Agreement]&#40;https://gist.github.com/samarabbas/7dcd41eb1d847e12263cc961ccfdb197&#41; before we can merge in any of your changes.)

## Prerequisites

### Build prerequisites
* [Go Lang](https://golang.org/) (minimum version required is 1.17):
    - Install on macOS with `brew install go`.
    - Install on Ubuntu with `sudo apt install golang`.

## Check out the code
Temporalite uses go modules, there is no dependency on `$GOPATH` variable. Clone the repo into the preferred location:
```bash
git clone https://github.com/DataDog/temporalite.git
```

## Build
Build the `temporalite` binary:
```bash
go build ./cmd/temporalite
```

## Run tests
Run all tests:
```bash
go test ./...
```

## Run Temporalite locally
Run the server in ephemeral mode:
```bash
go run ./cmd/temporalite start --ephemeral
```

Now you can create default namespace with `tctl`:
```bash
tctl --ns default namespace register
```
and run samples from [Go](https://github.com/temporalio/samples-go) and [Java](https://github.com/temporalio/samples-java) samples repos.

When you are done, press `Ctrl+C` to stop the server.

## License headers
This project is Open Source Software, and requires a header at the beginning of
all source files. To verify that all files contain the header execute:
```bash
go run ./internal/copyright
```

## Third party code
The license, origin, and copyright of all third party code is tracked in `LICENSE-3rdparty.csv`.
To verify that this file is up to date execute:
```bash
go run ./internal/licensecheck
```
