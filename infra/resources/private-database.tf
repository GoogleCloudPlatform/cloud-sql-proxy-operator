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


resource "random_id" "private_db_name_suffix" {
  byte_length = 4
}

# See versions at https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/sql_database_instance#database_version
resource "google_sql_database_instance" "private_postgres" {
  provider = google-beta

  name             = "privateinst${random_id.private_db_name_suffix.hex}${var.environment_name}"
  project          = var.project_id
  region           = var.gcloud_region
  database_version = "POSTGRES_13"
  depends_on       = [
    google_service_networking_connection.private_vpc_connection
  ]

  settings {
    tier        = "db-f1-micro"
    user_labels = local.standard_labels
    ip_configuration {
      ipv4_enabled                                  = false
      private_network                               = google_compute_network.private_k8s_network.id
      enable_private_path_for_google_cloud_services = true
    }
  }
  deletion_protection = "true"
  root_password       = random_id.db_password.hex
}

resource "google_sql_database" "private_db" {
  name     = "db"
  instance = google_sql_database_instance.private_postgres.name
  project  = var.project_id
}

resource "google_alloydb_cluster" "private_alloydb" {
  cluster_id = "alloy-${var.environment_name}"
  location   = "us-central1"
  network    = "projects/${data.google_project.project.number}/global/networks/${google_compute_network.private_k8s_network.name}"
  project    = data.google_project.project.project_id
  labels     = {} //TODO refresh bug empty is ok

  initial_user {
    password = random_id.db_password.hex
  }

  // TODO: refresh bug hardcoded defaults not respected
  automated_backup_policy {
    backup_window = "3600s"
    enabled       = true
    labels        = {}
    location      = "us-central1"
    time_based_retention {
      retention_period = "1209600s"
    }
    weekly_schedule {
      # forces replacement
      days_of_week = [
        "MONDAY",
        "TUESDAY",
        "WEDNESDAY",
        "THURSDAY",
        "FRIDAY",
        "SATURDAY",
        "SUNDAY",
      ]
      start_times {
        hours   = 23
        minutes = 0
        nanos   = 0
        seconds = 0
      }
    }
  }

}

resource "google_alloydb_instance" "private_alloydb" {
  cluster       = google_alloydb_cluster.private_alloydb.cluster_id
  instance_type = "PRIMARY"
  instance_id   = "dbinstance"

  machine_config {
    cpu_count = 2
  }

  depends_on = [google_service_networking_connection.private_vpc_connection]
}

output "private_db_root_password" {
  value = random_id.db_password.hex
}
output "private_db_instance_name" {
  value = google_sql_database_instance.private_postgres.name
}
output "private_db_database_name" {
  value = google_sql_database.private_db.name
}

output "alloy_db_cluster_id" {
  value = google_alloydb_cluster.private_alloydb.cluster_id
}

#output "alloy_db_instance_id" {
#  value = google_alloydb_instance.private_alloydb.instance_id
#}
