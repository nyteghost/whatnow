# syntax=docker/dockerfile:1

# The web client, built into the directory the Go binary embeds.
FROM --platform=$BUILDPLATFORM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json web/.npmrc ./
RUN npm ci
COPY web/ ./
RUN npm run build

# One static binary. modernc's SQLite is pure Go, so no cgo, and it cross-compiles for arm64.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
ARG TARGETOS TARGETARCH VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/build/ internal/web/dist/
# timetzdata embeds the zone database, so TZ works in an image without one.
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -tags timetzdata \
      -ldflags="-s -w -X main.Version=${VERSION}" -o /out/whatnow ./cmd/whatnow \
 && mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/whatnow /usr/local/bin/whatnow
COPY --from=build --chown=nonroot:nonroot /out/data /data
# There is no keychain in a container, so keys and integration secrets go in the volume.
ENV WHATNOW_SECRETS_FILE=/data/secrets.json \
    TZ=UTC
VOLUME /data
EXPOSE 7777
# Listening beyond loopback is safe only because the server still refuses any Host not allowed
# by WHATNOW_ALLOWED_HOSTS or in settings.
ENTRYPOINT ["whatnow", "serve", "-addr", "0.0.0.0:7777", "-db", "/data/whatnow.db"]
