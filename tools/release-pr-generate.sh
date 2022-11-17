#!/usr/bin/env bash
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
PROJECT_DIR=$( dirname "$SCRIPT_DIR")

cd "$PROJECT_DIR"

make generate

if git diff --exit-code ; then
  echo "Generate did not cause any changes to the code. OK to proceed with the release"
else
  echo "Generate updated the code. Committing the changes..."
  git add .
  git commit -m "chore: ensure that code is consistent using make generate"
  git push
  echo "OK to proceed with the release."
fi
