# Actual ARGS must be passed during docker build
ARG BASE_IMAGE_TAG=latest


FROM golang:${BASE_IMAGE_TAG} AS base

ARG TASKFILE_VERSION=latest

# Manually set $GOCACHE to /go/pkg/cache because $GOCACHE defaults to /root/.cache/go-build. This way, we don't have to mount both /go and /root/.cache directories during build to use as cache.
ENV GOPATH=/go GOCACHE=/go/pkg/cache

WORKDIR /app

RUN apk add --no-cache curl

# Development stage

FROM base AS development

RUN go install github.com/go-task/task/v3/cmd/task@${TASKFILE_VERSION}

COPY taskfile.yaml .

RUN task init

COPY go.mod go.sum ./

RUN go mod download -x

ENTRYPOINT [ "task", "dev" ]

# Release builder stage

FROM base AS release-builder

ARG TASKFILE_VERSION=latest

ARG BINARY_PATH

RUN go install github.com/go-task/task/v3/cmd/task@${TASKFILE_VERSION}

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/go/pkg/cache task build:release && mv ${BINARY_PATH} /app/main

# Release stage

FROM gcr.io/distroless/base-debian12:nonroot AS release

COPY --from=release-builder /app/main /main

ENTRYPOINT ["/main"]