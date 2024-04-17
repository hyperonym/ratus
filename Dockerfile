# syntax=docker/dockerfile:1

# Set default Go version
ARG GO_VERSION=1.22.2

# Use native architecture of the build node for cross-compilation
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build

# Set working directory
WORKDIR /src

# Install module dependencies
COPY go.mod .
COPY go.su[m] .
RUN go mod download -x

# Copy source code
COPY . .

# Build binaries for the target platform
ARG TARGETOS
ARG TARGETARCH
ARG VERSION
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o bin/ ./cmd/*

# Copy binaries from the build stage for exporting
FROM --platform=$TARGETPLATFORM scratch AS binary
COPY --from=build /src/bin/* /

# Build base image with binaries and trusted certificates
FROM --platform=$TARGETPLATFORM scratch AS base
COPY --from=binary /* /
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Build release image with configurations
FROM base AS release

# Expose ports
EXPOSE 80

# Provide default environment variables
ENV GIN_MODE="release"
ENV ENGINE="memdb"
ENV PORT="80"
ENV ADDR="0.0.0.0"
ENV CHORE_INTERVAL="10s"
ENV CHORE_INITIAL_DELAY="10s"
ENV CHORE_INITIAL_RANDOM="true"
ENV PAGINATION_MAX_LIMIT="100"
ENV PAGINATION_MAX_OFFSET="10000"
ENV MEMDB_SNAPSHOT_PATH="/ratus.db"
ENV MEMDB_SNAPSHOT_INTERVAL="5m"
ENV MEMDB_RETENTION_PERIOD="72h"
ENV MONGODB_URI="mongodb://mongo:27017"
ENV MONGODB_DATABASE="ratus"
ENV MONGODB_COLLECTION="tasks"
ENV MONGODB_RETENTION_PERIOD="72h"
ENV MONGODB_DISABLE_INDEX_CREATION="false"
ENV MONGODB_DISABLE_AUTO_FALLBACK="false"
ENV MONGODB_DISABLE_ATOMIC_POLL="false"

# Specify entrypoint and default parameters
ENTRYPOINT ["/ratus"]
