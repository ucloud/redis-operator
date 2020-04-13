FROM golang:1.13.3-alpine as go-builder

RUN apk update && apk upgrade && \
    apk add --no-cache ca-certificates git mercurial

ARG PROJECT_NAME=redis-operator
ARG REPO_PATH=github.com/ucloud/$PROJECT_NAME
ARG BUILD_PATH=${REPO_PATH}/cmd/manager

# Build version and commit should be passed in when performing docker build
ARG VERSION=0.1.1
ARG GIT_SHA=0000000

WORKDIR /src

COPY go.mod go.sum ./
RUN GOPROXY=https://goproxy.cn,direct go mod download

COPY ./ ./

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ${GOBIN}/${PROJECT_NAME} \
    -ldflags "-X ${REPO_PATH}/pkg/version.Version=${VERSION} -X ${REPO_PATH}/pkg/version.GitSHA=${GIT_SHA}" \
    $BUILD_PATH

# =============================================================================
FROM alpine:3.9 AS final

ARG PROJECT_NAME=redis-operator

COPY --from=go-builder ${GOBIN}/${PROJECT_NAME} /usr/local/bin/${PROJECT_NAME}

RUN adduser -D ${PROJECT_NAME}
USER ${PROJECT_NAME}

ENTRYPOINT ["/usr/local/bin/redis-operator"]
