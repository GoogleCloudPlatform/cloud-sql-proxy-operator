#!/usr/bin/env bash
# Copyright 2023 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


function trigger_build() {
    set -x
    # Use --billing-project flag until WIF project enables the cloud build API
    gcloud builds triggers run e2e-test-pr-manual \
      --sha "$commitsha" \
      --billing-project "$project" \
      --project="$project" \
      --format="value(metadata.build.id)" > build-id.txt
    set +x
    echo
    echo "View build logs in Google Cloud Console: "
    gcloud builds describe "$build_id" \
        --project "$project" \
        --billing-project "$project" \
        --format="value(logUrl)"
}

function wait_for_build() {
    build_id=$(cat build-id.txt)
    echo "Build ID: $build_id"
    echo
    echo "This Github Action will now poll for build completion..."
    echo

    # Wait for build to finish
    while true ; do
      gcloud builds describe "$build_id" \
        --project "$project" \
        --billing-project "$project" \
        --format="value(status)" > status.txt

      s=$(cat status.txt)

      echo "Build Status $(date '+%Y-%m-%dT%H:%M:%S%z') $s"

      if [[ $s == "QUEUED" || $s == "WORKING" ]] ; then
        sleep 30
      elif [[ $s == "SUCCESS" ]] ; then
        exit 0
      elif [[ $s == "CANCELLED" ]] ; then
        echo "The Cloud Build job was canceled."
        exit 1
      else
        echo "Build failed"
        exit 1
      fi
    done
}

case $1 in
trigger_build)
  trigger_build
  ;;
wait_for_build)
  wait_for_build
  ;;
*)
  echo "Bad command: [trigger_build|wait_for_build]"
  exit 1
esac