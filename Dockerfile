# syntax=docker/dockerfile:1
ARG BASE_IMAGE_TAG

# Base image to avoid repeating common setup
FROM golang${BASE_IMAGE_TAG} AS base

WORKDIR /app

RUN apk add --no-cache build-base bash git

# Development image
FROM base AS development

# Install Air for live reload
RUN go install github.com/air-verse/air@latest

# Use BuildKit's mount feature for efficient dependency resolution
RUN --mount=type=bind,source=go.mod,target=go.mod,readonly \
    --mount=type=bind,source=go.sum,target=go.sum,readonly \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download -x

ENTRYPOINT ["./run", "watch"]

# Production builder image
FROM base AS production-builder

ENV GOPATH=/go
ENV GOCACHE=/root/.cache/go-build

# Efficient dependency resolution with caching
RUN --mount=type=bind,source=go.mod,target=go.mod,readonly \
    --mount=type=bind,source=go.sum,target=go.sum,readonly \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download -x && go mod verify

COPY . .

# Build the application with caching for compiled objects
RUN --mount=type=cache,target=${GOCACHE} ./run build

# Final production image
FROM scratch AS production

WORKDIR /app

# Copy only the built binary
COPY --from=production-builder /app/bin .

ENTRYPOINT ["./main"]
