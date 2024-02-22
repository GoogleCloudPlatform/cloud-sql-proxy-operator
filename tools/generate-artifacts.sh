#!/usr/bin/env bash
# Copyright 2024 Google LLC
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
GOLANG_VERSION=1.20

cd "$PROJECT_DIR"

apt-get -yq update 
apt-get -yq install unzip wget

wget https://go.dev/dl/go$GOLANG_VERSION.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go$GOLANG_VERSION.linux-amd64.tar.gz
rm go$GOLANG_VERSION.linux-amd64.tar.gz
export PATH="${PATH}:/usr/local/go/bin"

make install_tools
make generate
