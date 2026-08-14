# syntax=docker/dockerfile:1
# Local / CI image: compile inside Docker (e.g. make docker-build, docker compose).
# Release images: GoReleaser builds static binaries, then Dockerfile.release packages them.
# Multi-arch: docker buildx build --platform linux/amd64,linux/arm64 -t groot:local .
# Makefile passes APP_VERSION, GIT_COMMIT, GIT_BRANCH, BUILD_DATE from VERSION, git, and date.
FROM --platform=$BUILDPLATFORM golang:1.26.6-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG APP_VERSION=0.0.0
ARG GIT_COMMIT=unknown
ARG GIT_BRANCH=unknown
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath \
	-ldflags="-s -w -X main.version=${APP_VERSION} -X main.commit=${GIT_COMMIT} -X main.branch=${GIT_BRANCH} -X main.buildDate=${BUILD_DATE}" \
	-o /out/groot ./cmd/groot

FROM gcr.io/distroless/static-debian13:nonroot
WORKDIR /app
COPY --from=builder /out/groot /app/groot
COPY configs/groot.yml.sample /app/groot.yml.sample
USER nonroot:nonroot
ENTRYPOINT ["/app/groot"]
CMD ["--help"]
