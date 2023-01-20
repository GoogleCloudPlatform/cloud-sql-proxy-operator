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

resource "google_artifact_registry_repository" "artifact_repo" {
  location      = var.gcloud_region
  repository_id = "test${var.environment_name}"
  description   = "Operator test artifact repo"
  format        = "DOCKER"
  project       = var.project_id
  labels        = local.standard_labels
}

// example: us-central1-docker.pkg.dev/csql-operator-test/test76e6d646e2caac1c458c
resource "local_file" "artifact_repo_url" {
  content = join("/", [
    "${google_artifact_registry_repository.artifact_repo.location}-docker.pkg.dev",
    google_artifact_registry_repository.artifact_repo.project,
  google_artifact_registry_repository.artifact_repo.name])
  filename = var.gcloud_docker_url_file
}
