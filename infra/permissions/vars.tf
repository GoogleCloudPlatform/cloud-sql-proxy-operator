/**
 * Copyright 2023 Google LLC
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
  description = "The name of the environment to create, a single gcp project can host many test environments"
}

variable "output_json_path" {
  type        = string
  description = "The path to save output.json file. This contains the values created by this project"
}

variable "gcloud_bin" {
  type        = string
  description = "The absolute path to the gcloud executable"
}

variable "gcloud_zone" {
  default = "us-central1-c"
}

variable "gcloud_region" {
  default = "us-central1"
}
