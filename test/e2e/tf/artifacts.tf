resource "google_artifact_registry_repository" "artifact_repo" {
  location      = var.gcloud_region
  repository_id = "test${random_id.cluster_name.hex}"
  description   = "Operator test artifact repo"
  format        = "DOCKER"
  project       = var.project_id
}

// us-central1-docker.pkg.dev/csql-operator-test/test76e6d646e2caac1c458c
resource "local_file" "artifact_repo_url" {
  #  content  = "${google_artifact_registry_repository.artifact_repo.location}-docker.pkg.dev/${google_artifact_registry_repository.artifact_repo.project}/${google_artifact_registry_repository.artifact_repo.name}"
  content = join("/", [
    "${google_artifact_registry_repository.artifact_repo.location}-docker.pkg.dev",
    google_artifact_registry_repository.artifact_repo.project,
  google_artifact_registry_repository.artifact_repo.name])
  filename = var.gcloud_docker_url_file
}
