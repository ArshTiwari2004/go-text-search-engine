
# Stage 1 → Build binary (heavy image)
# Stage 2 → Run binary (light image)

# multi-stage build → reduces image size drastically


# ── Stage 1: Build
FROM golang:1.21-alpine AS builder

# Install git for go modules that fetch from VCS
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy dependency files first, Docker layer cache means 'go mod download'
# only re-runs when go.mod/go.sum change, not on every code change.
COPY go.mod go.sum ./
# copy go.mod first so dependencies are cached and not re downloaded on every build
RUN go mod download

COPY . .

# CGO_ENABLED=0 — pure-Go binary, no libc dependency → works in scratch/alpine
# -ldflags="-s -w" — strip debug symbols → binary ~30% smaller
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o gosearch \
    ./cmd/api

# ── Stage 2: Runtime 
# Use alpine (not scratch) so we have:
#  - ca-certificates for HTTPS outbound calls
#  - wget for the Docker healthcheck
FROM alpine:3.19

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

# Copy only the compiled binary — no Go toolchain, no source code in the image
COPY --from=builder /app/gosearch .

# data/ directory will be mounted as a Docker volume for index persistence
RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./gosearch"]