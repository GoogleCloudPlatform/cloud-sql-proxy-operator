// Copyright 2022 Google LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package workloads holds logic for manipulating kubernetes workload
// data structs. Code in this package assumes that it is single-threaded, running
// on data structures only accessible to the current thread.
//
// In addition, workloads contains functions that help format safe names for
// // kubernetes resources, following the rfc1035/rfc1123 label (DNS_LABEL) format.
package workloads
