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

##
# build-identifier uses the HEAD Git SHA to provide a unique id number for a build.
# If the working directory is dirty, it will append the current timestamp
# to the HEAD Git SHA so that the build identifier is unique.
if [[ -n ${RELEASE_TEST_BUILD_ID:-} ]] ; then
  echo "${RELEASE_TEST_BUILD_ID}"
  exit 0
fi

if [[ -n ${KOKORO_GIT_COMMIT_connector_operator:-} ]] ; then
  echo -n "${KOKORO_GIT_COMMIT_connector_operator}"
  exit 0
fi

if [[ -n ${KOKORO_GIT_COMMIT:-} ]] ; then
  echo -n "${KOKORO_GIT_COMMIT}"
  exit 0
fi

NOW=$(date -u "+%Y%m%dT%H%M" | tr -d "\n")

if jj root >/dev/null 2>&1; then
  JJ_COMMIT=$(jj log -r @ -T "commit_id" --no-graph | tr -d "\n")
  if jj status | grep -q "Working copy changes:"; then
    IMAGE_VERSION="$JJ_COMMIT-dirty-${NOW}"
  else
    IMAGE_VERSION="$JJ_COMMIT"
  fi
else
  GIT_HEAD=$( git rev-parse HEAD | tr -d "\n" )
  if git diff HEAD --exit-code --quiet ; then
    IMAGE_VERSION="$GIT_HEAD"
  else
    IMAGE_VERSION="$GIT_HEAD-dirty-${NOW}"
  fi
fi

echo -n "$IMAGE_VERSION"
