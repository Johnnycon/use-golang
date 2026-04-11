# ==============================================================================
# Stage 1: Generate GraphQL code
# ==============================================================================
FROM golang:1.23-alpine AS gqlgen

RUN apk add --no-cache git

WORKDIR /app

# Copy files gqlgen needs — NOT server.go or schema.resolvers.go.
# resolver.go imports the db package, so we include it here too.
COPY go.mod tools.go gqlgen.yml ./
COPY graph/schema.graphqls graph/
COPY graph/resolver.go graph/
COPY graph/model/model.go graph/model/
COPY db/ db/

# Strip deps not needed in the gqlgen stage (e.g. River) so go mod tidy
# doesn't fail trying to resolve them.
RUN sed -i '/riverqueue/d; /openai/d; /google.golang.org\/genai/d' go.mod && go mod tidy
RUN go run github.com/99designs/gqlgen generate

# ==============================================================================
# Stage 2: Build the binary
# ==============================================================================
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /app

COPY . .
COPY --from=gqlgen /app/graph/generated.go graph/generated.go
COPY --from=gqlgen /app/graph/model/models_gen.go graph/model/
COPY --from=gqlgen /app/go.sum .

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /api .

# ==============================================================================
# Stage 3: Minimal runtime image
# ==============================================================================
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /api /api

EXPOSE 8080

ENTRYPOINT ["/api"]
