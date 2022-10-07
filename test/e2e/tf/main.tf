terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "4.31.0"
    }
  }
}


provider "google" {
}

# Enable gcloud project APIs
locals {
  project_services = toset([
    "compute.googleapis.com",
    "container.googleapis.com", # GKE
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
    "sql-component.googleapis.com",
    "sqladmin.googleapis.com",
  "storage-api.googleapis.com"])
}

resource "google_project_service" "project" {
  for_each = local.project_services
  project  = var.project_id
  service  = each.value
}


##
# This is how you do an output file containing terraform data for use by
# a subsequent script.

# First, create the output data structure as a local variable
locals {
  testinfra = {
    instance     = google_sql_database_instance.instance.connection_name
    db           = google_sql_database.db.name
    rootPassword = random_id.db_password.hex
    kubeconfig   = var.kubeconfig_path
  }
}

# Then write the output data to a local file in json format
resource "local_file" "testinfra" {
  content  = jsonencode(local.testinfra)
  filename = var.testinfra_json_path
}
