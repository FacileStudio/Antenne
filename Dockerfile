# The client is built first and copied into the API image, so one container
# serves the API, the webhook receivers, the pool and the dashboard.
FROM oven/bun:1 AS client-build
WORKDIR /client
COPY apps/client/package.json apps/client/bun.lock* ./
RUN bun install --frozen-lockfile
COPY apps/client/ .
RUN bun run build

FROM golang:1.26-alpine AS api-build

ARG TARGETOS=linux
ARG TARGETARCH
WORKDIR /repo/apps/api

COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api/ .
COPY CHANGELOG.md /version/CHANGELOG.md
COPY scripts/version.sh /version/version.sh

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w -X main.version=$(sh /version/version.sh /version/CHANGELOG.md)" -o /out/antenne .

# :nonroot, because the binary writes nothing to disk — every byte of state is in
# Postgres. The root variant was only needed while a legacy settings file had to
# be read off a root-owned named volume, and that volume is gone.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=api-build /out/antenne /antenne
COPY --from=client-build /client/build /client

# Obligatory: a relative ./client would resolve against the image's working
# directory, and the dashboard would silently not be served — the API answers,
# the healthcheck is green, and only a browser sees the failure.
ENV CLIENT_DIR=/client

EXPOSE 9090

USER nonroot:nonroot

ENTRYPOINT ["/antenne"]
