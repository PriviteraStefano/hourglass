# syntax=docker/dockerfile:1

# --- build stage ---
FROM golang:1.26.1-alpine AS build
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# pgx is pure Go → fully static binaries, no CGO
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/hourglass ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# --- runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/hourglass /out/migrate /usr/local/bin/
# migrations are read from disk (not embed.FS)
COPY --from=build /src/migrations /migrations
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/hourglass"]
