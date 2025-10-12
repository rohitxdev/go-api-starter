# Actual base image tag must be passed during docker build
ARG BASE_IMAGE_TAG=latest

FROM golang:${BASE_IMAGE_TAG} AS base

# Manually set $GOCACHE to /go/pkg/cache because $GOCACHE defaults to /root/.cache/go-build. This way, we don't have to mount both /go and /root/.cache directories during build to use as cache.
ENV GOCACHE=/go/pkg/cache

WORKDIR /app

RUN apk add --no-cache curl

# Development stage

FROM base AS development

RUN go install github.com/go-task/task/v3/cmd/task@latest

COPY taskfile.yaml .

RUN task init

COPY go.mod go.sum ./

RUN go mod download -x

ENTRYPOINT [ "task", "dev" ]

# Release builder stage

FROM base AS release-builder

ARG BINARY_PATH

COPY . .

RUN --mount=type=cache,target=/go go install github.com/go-task/task/v3/cmd/task@latest && task build:release

RUN mv ${BINARY_PATH} ./main

# Release stage

FROM scratch AS release

COPY --from=release-builder /app/main .

ENTRYPOINT ["./main"]