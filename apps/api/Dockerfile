# Multi-stage Dockerfile for the Alethea API.
# - `dev` target: hot-reloads via `go run` against a bind-mounted /src
# - `prod` target: static binary in a distroless image (~10MB)
# Choose the target at build time via docker-compose `target:`.

# ---------- base deps ----------
FROM golang:1.21-alpine AS deps
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download

# ---------- dev (compose default) ----------
FROM deps AS dev
WORKDIR /src
COPY . .
EXPOSE 8080
# Prompts are go:embed'd so changes to assets/ require rebuild — fine for dev.
CMD ["go", "run", "."]

# ---------- prod build ----------
FROM deps AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/alethea-api .

# ---------- prod runtime ----------
FROM gcr.io/distroless/static-debian12 AS prod
COPY --from=build /out/alethea-api /alethea-api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/alethea-api"]
