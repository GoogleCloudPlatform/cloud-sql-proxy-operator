variable "gcloud_project_id" {
  type = string
  description = "The gcloud project id"
}

variable "kubeconfig_path" {
  type = string
  description = "The path to save the kubeconfig file"
}
variable "testinfra_json_path" {
  type = string
  description = "The path to save test-infra.json file, input for e2e tests"
}
variable "gcloud_docker_url_file" {
  type = string
  description = "The path to save the artifact repo url"
}
variable "gcloud_bin" {
  type = string
  description = "The absolute path to the gcloud executable"
}
