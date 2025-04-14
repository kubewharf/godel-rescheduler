#!/bin/bash
# error on exit
set -e

# verbose for debugging
set -x

DOCKERFILE_DIR="docker/"

docker container prune -f || true;
docker image prune -f || true

for file in $(find "$DOCKERFILE_DIR" -name *.Dockerfile);
do
  REPO=$(basename $file);
  REPO=${REPO%.*};
  REV=$(git log --pretty=format:'%h' -n 1);
  TAG=${REPO}:${REV}
  docker rmi "${TAG}" || true;
  docker rmi "${REPO}:latest" || true
  docker build -t "${TAG}"  -f $file ./;
  docker tag "${TAG}" "${REPO}:latest"
done;
