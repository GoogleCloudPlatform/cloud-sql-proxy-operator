/**
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

variable "project_id" {
  type        = string
  description = "The gcloud project id"
}

variable "environment_name" {
  type        = string
  description = "The test environment name"
}

variable "kubeconfig_path" {
  type        = string
  description = "The path to save the kubeconfig file"
}
variable "output_json_path" {
  type        = string
  description = "The path to save test-infra.json file, input for e2e tests"
}
variable "gcloud_docker_url_file" {
  type        = string
  description = "The path to save the artifact repo url"
}
variable "gcloud_bin" {
  type        = string
  description = "The absolute path to the gcloud executable"
}

variable "nodepool_serviceaccount_email" {
  description = "The service account email address to assign to the nodepool"
}

variable "kubernetes_version" {
  default = ""
}

variable "workers_count" {
  default = "2"
}

variable "node_machine_type" {
  default = "e2-standard-2"
}

variable "enable_alpha" {
  default = false
}

variable "gcloud_zone" {
  default = "us-central1-c"
}
variable "gcloud_region" {
  default = "us-central1"
}
