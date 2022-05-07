FROM golang:1.18 as builder

WORKDIR ${GOPATH:-/go}/src/temporalite

COPY . .
RUN go mod download
RUN go get -d -v ./...

RUN go build -o ${GOPATH:-/go}/bin/ ${GOPATH:-/go}/src/temporalite/cmd/temporalite

FROM gcr.io/distroless/base-debian11

COPY --from=builder ${GOPATH:-/go}/bin/temporalite /
EXPOSE 7233

ENTRYPOINT ["/temporalite", "start", "--ephemeral", "-n", "default", "--ip" , "0.0.0.0"]
