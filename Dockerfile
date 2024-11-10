ARG BASE_IMAGE_TAG

# Development image
FROM golang${BASE_IMAGE_TAG} AS development

WORKDIR /app

RUN apk add --no-cache build-base bash git && go install github.com/air-verse/air@latest

RUN --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

ENTRYPOINT ["./run","watch"]


# Production builder image
FROM golang${BASE_IMAGE_TAG} AS production-builder

WORKDIR /app

RUN apk add --no-cache build-base bash git

ENV GOPATH=/go

RUN --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download -x && go mod verify

COPY . .

ENV GOCACHE=/root/.cache/go-build

RUN --mount=type=cache,target=${GOCACHE} ./run build


# Final production image
FROM scratch AS production

WORKDIR /app

COPY --from=production-builder /app/bin .

ENTRYPOINT ["./main"]