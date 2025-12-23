# syntax=docker/dockerfile:1.7

########################
# 1) Build stage
########################
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE
ENV DATE=${DATE}

WORKDIR /src

# Cache deps
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy sources
COPY . .

# Build static binary
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -v -trimpath -buildvcs=false \
      -ldflags="-s -w \
        -X github.com/Chapsvision-dev/vault-raft-backup-restore/internal/version.Version=${VERSION} \
        -X github.com/Chapsvision-dev/vault-raft-backup-restore/internal/version.Commit=${COMMIT} \
        -X github.com/Chapsvision-dev/vault-raft-backup-restore/internal/version.BuildDate=${DATE}" \
      -o /out/operator ./cmd/operator && \
      mkdir -p /workspace

########################
# 2) Runtime stage
########################
FROM scratch AS runtime

# OCI labels
LABEL org.opencontainers.image.title="vault-raft-backup-operator"
LABEL org.opencontainers.image.description="Backup and restore Vault Hashicorp Raft snapshots to cloud storage"
LABEL org.opencontainers.image.url="https://github.com/Chapsvision-dev/vault-raft-backup-restore"
LABEL org.opencontainers.image.source="https://github.com/Chapsvision-dev/vault-raft-backup-restore"
LABEL org.opencontainers.image.documentation="https://github.com/Chapsvision-dev/vault-raft-backup-restore#readme"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.vendor="Chapsvision"

# Copy CA certs (needed for HTTPS/Azure SDK)
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=build /out/operator /operator

# Run as non-root
USER 65532:65532

# Zerolog defaults
ENV LOG_LEVEL=info \
    LOG_FORMAT=json \
    TZ=Etc/UTC

ENTRYPOINT ["/operator"]
CMD ["--help"]
