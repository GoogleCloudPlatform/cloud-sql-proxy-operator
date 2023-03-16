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

terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "4.56.0"
    }
  }
}


provider "google" {
  user_project_override = true
  billing_project       = var.project_id
}

# Enable gcloud project APIs
locals {
  project_services = toset([
    "compute.googleapis.com",
    "container.googleapis.com",
    "artifactregistry.googleapis.com",
    "deploymentmanager.googleapis.com",
    "dns.googleapis.com",
    "logging.googleapis.com",
    "monitoring.googleapis.com",
    "oslogin.googleapis.com",
    "pubsub.googleapis.com",
    "replicapool.googleapis.com",
    "replicapoolupdater.googleapis.com",
    "resourceviews.googleapis.com",
    "servicemanagement.googleapis.com",
    "servicenetworking.googleapis.com",
    "sql-component.googleapis.com",
    "sqladmin.googleapis.com",
  "storage-api.googleapis.com"])
}

resource "google_project_service" "project" {
  for_each = local.project_services
  project  = var.project_id
  service  = each.value
}

# Create service accounts for k8s workload nodes
resource "google_service_account" "node_pool" {
  account_id   = "k8s-nodes-${var.environment_name}"
  display_name = "Kubernetes provider SA"
  project      = var.project_id
}
resource "google_project_iam_member" "allow_image_pull" {
  project = var.project_id
  role    = "roles/artifactregistry.reader"
  member  = "serviceAccount:${google_service_account.node_pool.email}"
}

resource "google_project_iam_binding" "cloud_sql_client" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  members = [
    "serviceAccount:${google_service_account.node_pool.email}"
  ]
}

##
# This is how you do an output file containing terraform data for use by
# a subsequent script.

# First, create the output data structure as a local variable
locals {
  tf_output = {
    project_id                    = var.project_id
    environment_name              = var.environment_name
    nodepool_serviceaccount_email = google_service_account.node_pool.email
  }
}

# Then write the output data to a local file in json format
resource "local_file" "tf_output" {
  content  = jsonencode(local.tf_output)
  filename = var.output_json_path
}
