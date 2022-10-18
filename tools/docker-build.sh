#!/bin/bash

# Copyright 2022 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euxo
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

##
# build a docker image and tag it using
# Pass script params as environment variables:
function usage() {
cat <<EOF

docker-build.sh - generate a tag for a docker image from its git commit hash
   and run \`docker buildx build\`. Script parameters are passed in environment
   variables.

  PROJECT_DIR - location of the working directory
  IMAGE_NAME - the short name for the container image
  REPO_URL - the url to the container repository
  PLATFORMS - the docker buildx --platform argument
  DOCKER_FILE_NAME - (optional) relative filename of the Dockerfile from PROJECT_DIR
  IMAGE_URL_OUT - (optional) Write the docker image url into this file
  EXTRA_TAGS - (optional) Tag this image with this comma-separated list of tags
  LOAD - when set, script will run docker buildx with --load instead of --push

Usage:
  PROJECT_DIR=/home/projects/cloudsql/cloudsql-operator \\
  IMAGE_NAME=cloudsql-operator \\
  REPO_URL=uscentral-1.gcr.io/project/reponame \\
  IMAGE_URL_OUT=/home/projects/cloudsql/cloudsql-operator/bin/image-url.txt \\
  EXTRA_TAGS=us.gcr.io/repo:1,asia.gcr.io/repo:1.0
  PLATFORMS=linux/arm64/v8,linux/amd64 \\
  DOCKER_FILE_NAME=Dockerfile \\
  docker-build.sh

EOF
}

bad=""
if [[ -z "${PROJECT_DIR:-}" ]] ; then
  bad="bad"
  echo "PROJECT_DIR environment variable must be set"
fi
if [[ -z "${IMAGE_NAME:-}" ]] ; then
  bad="bad"
  echo "IMAGE_NAME environment variable must be set"
fi
if [[ -z "${REPO_URL:-}" && -z "${LOAD:-}" ]] ; then
  bad="bad"
  echo "either REPO_URL or LOAD environment variables must be set"
fi
if [[ -z "${PLATFORMS:-}" ]] ; then
  bad="bad"
  echo "PLATFORMS environment variable must be set"
fi
if [[ "${bad:-}" == "bad" ]] ; then
  echo
  usage
  exit 1
fi

set -x

cd "$PROJECT_DIR"

if [[ -z ${IMAGE_VERSION:-} ]] ; then
  IMAGE_VERSION=$( "$SCRIPT_DIR/build-identifier.sh" | tr -d '\n' )
fi

if [[ -z "${LOAD:-}" ]] ; then
  IMAGE_URL="${REPO_URL}/${IMAGE_NAME}:${IMAGE_VERSION}"
  LOAD_ARG="--push"
else
  IMAGE_URL="${IMAGE_NAME}:${IMAGE_VERSION}"
  LOAD_ARG="--load"
fi

TAG_FLAGS=("-t" "$IMAGE_URL")
if [[ -n "${EXTRA_TAGS:-}" ]] ; then
  # split EXTRA_TAGS by space and comma
  IFS=', ' read -a extra_tags <<< "${EXTRA_TAGS}"
  TAG_FLAGS=()
  for tag in "${extra_tags[@]}"
  do
      TAG_FLAGS+=("-t" "$tag")
  done
fi

docker buildx build --platform "$PLATFORMS" \
  -f "${DOCKER_FILE_NAME:-Dockerfile}" \
   "${TAG_FLAGS[@]}" \
  "$LOAD_ARG" "$PWD"

if [[ -n "${IMAGE_URL_OUT:-}" ]] ; then
  mkdir -p $(dirname "$IMAGE_URL_OUT")
  echo "Writing image url to $IMAGE_URL_OUT"
  echo -n "$IMAGE_URL" > "$IMAGE_URL_OUT"
fi

echo "Docker buildx build complete."
