# syntax=docker/dockerfile:1

FROM golang:1.24-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/ozsh ./cmd/ozsh

FROM debian:bookworm-slim AS runtime

RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates zsh \
  && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/ozsh /usr/local/bin/ozsh

RUN useradd -m -u 10001 ozsh
USER ozsh
WORKDIR /home/ozsh

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 CMD ["/usr/local/bin/ozsh", "version"]

ENTRYPOINT ["/usr/local/bin/ozsh"]
CMD ["version"]
