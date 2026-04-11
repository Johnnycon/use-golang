# ==============================================================================
# Stage 1: Build the Go binary
# ==============================================================================
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Cache dependencies layer — only re-downloads when go.mod changes.
COPY go.mod ./
RUN go mod download

# Copy everything (including templates/ which gets embedded via go:embed).
COPY . .

# Static binary, stripped of debug symbols.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /web .

# ==============================================================================
# Stage 2: Minimal runtime image
# ==============================================================================
# We use scratch here. The templates are embedded in the binary at compile time
# (go:embed), so no extra files are needed at runtime.
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /web /web

EXPOSE 3080

ENTRYPOINT ["/web"]
