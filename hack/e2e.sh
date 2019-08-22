#!/bin/bash

set -e

readonly REPO_PATH=github.com/ucloud/redis-operator

if [[ -z ${STORAGECLASSNAME} ]]; then
    echo "env STORAGECLASSNAME not set"
    exit 1
fi

if [[ -z ${KUBECONFIG} ]]; then
    echo "env KUBECONFIG not set"
    exit 1
fi

if [[ -z ${KUNE2E_GINKGO_SKIP} ]]; then
    export KUNE2E_GINKGO_SKIP=""
fi

echo "run e2e tests..."
echo "cd ${GOPATH}/src/${REPO_PATH} && ginkgo -v --skip=$KUNE2E_GINKGO_SKIP test/e2e/rediscluster"
cd ${GOPATH}/src/${REPO_PATH} && ginkgo -v --skip=${KUNE2E_GINKGO_SKIP} test/e2e/rediscluster

