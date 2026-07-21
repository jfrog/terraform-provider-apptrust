#!/usr/bin/env bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"
source "${SCRIPT_DIR}/get-access-key.sh"
source "${SCRIPT_DIR}/wait-for-rt.sh"

export ARTIFACTORY_VERSION=${ARTIFACTORY_VERSION:-7.146.15}
echo "ARTIFACTORY_VERSION=${ARTIFACTORY_VERSION}" > /dev/stderr

set -euf

rm -rf ${SCRIPT_DIR}/artifactory/

mkdir -p ${SCRIPT_DIR}/artifactory/extra_conf
mkdir -p ${SCRIPT_DIR}/artifactory/var/etc

cp ${SCRIPT_DIR}/artifactory.lic ${SCRIPT_DIR}/artifactory/extra_conf
cp ${SCRIPT_DIR}/system.yaml ${SCRIPT_DIR}/artifactory/var/etc/

echo "Starting PostgreSQL container"
docker run -i --name postgres -d --rm \
  -e POSTGRES_DB=artifactory \
  -e POSTGRES_USER=artifactory \
  -e POSTGRES_PASSWORD=password \
  -p 5432:5432 \
  postgres:16-alpine

echo "Waiting for PostgreSQL to be ready"
until docker exec postgres pg_isready -U artifactory; do
    printf '.'
    sleep 2
done
echo "PostgreSQL is ready"

echo "Starting Artifactory container"
docker run -i --name artifactory -d --rm \
  --add-host=host.docker.internal:host-gateway \
  -v ${SCRIPT_DIR}/artifactory/extra_conf:/artifactory_extra_conf \
  -v ${SCRIPT_DIR}/artifactory/var:/var/opt/jfrog/artifactory \
  -p 8081:8081 -p 8082:8082 \
  releases-docker.jfrog.io/jfrog/artifactory-pro:${ARTIFACTORY_VERSION}

export ARTIFACTORY_URL=http://localhost:8081
export ARTIFACTORY_UI_URL=http://localhost:8082

waitForArtifactory "${ARTIFACTORY_URL}" "${ARTIFACTORY_UI_URL}"

echo "export JFROG_ACCESS_TOKEN=$(getAccessKey "${ARTIFACTORY_UI_URL}")"
