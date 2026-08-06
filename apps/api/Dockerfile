# =============================================================================
# Stage 1: Build
# =============================================================================
FROM golang:1.25.2-alpine AS builder

# Install ca-certificates for HTTPS and tzdata for timezone support
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /build

# Copy go.work and all module files first for layer caching
COPY go.work go.work.sum ./
COPY apps/api/go.mod apps/api/go.sum ./apps/api/
COPY pkg/go.mod pkg/go.sum ./pkg/

# Download dependencies (cached unless go.mod/go.sum change)
RUN cd apps/api && go mod download
RUN cd pkg && go mod download

# Copy source code
COPY apps/api/ ./apps/api/
COPY pkg/ ./pkg/

# Build the binary with optimizations
RUN cd apps/api && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -ldflags="-w -s" \
      -trimpath \
      -o /build/nexus-api \
      ./cmd/server

# =============================================================================
# Stage 2: Runtime (distroless — minimal attack surface)
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot

# Copy timezone data and CA certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the compiled binary
COPY --from=builder /build/nexus-api /nexus-api

EXPOSE 8080

ENTRYPOINT ["/nexus-api"]