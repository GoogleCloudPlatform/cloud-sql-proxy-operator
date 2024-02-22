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
  depends_on = [
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
  location   = var.gcloud_region
  network    = "projects/${data.google_project.project.number}/global/networks/${google_compute_network.private_k8s_network.name}"
  project    = data.google_project.project.project_id
  labels     = {} //TODO file refresh bug empty is ok

  initial_user {
    password = random_id.db_password.hex
  }

  // TODO: file refresh bug hardcoded defaults not respected
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


locals {
  # Since it's an empty postgres instance, the only db name is "postgres"
  alloydb_db_name = "postgres"

  # The instance name created by hand is "db" See bug below.
  alloydb_instance_id = "db"

  # The instance URI formatted in accordance with documentation here:
  # https://cloud.google.com/alloydb/docs/auth-proxy/connect
  alloydb_instance_uri = "${data.google_project.project.id}/locations/${var.gcloud_region}/clusters/${google_alloydb_cluster.private_alloydb.cluster_id}/instances/${local.alloydb_instance_id}"
}

//TODO File bug, this doesn't even work.
//
//│ Error: Error creating Instance: googleapi: got HTTP response code 404 with body: <!DOCTYPE html>
//│ <html lang=en>
//│   <meta charset=utf-8>
//│   <meta name=viewport content="initial-scale=1, minimum-scale=1, width=device-width">
//│   <title>Error 404 (Not Found)!!1</title>
//│   <style>
//│     *{margin:0;padding:0}html,code{font:15px/22px arial,sans-serif}html{background:#fff;color:#222;padding:15px}body{margin:7% auto 0;max-width:390px;min-height:180px;padding:30px 0 15px}* > body{background:url(//www.google.com/images/errors/robot.png) 100% 5px no-repeat;padding-right:205px}p{margin:11px 0 22px;overflow:hidden}ins{color:#777;text-decoration:none}a img{border:0}@media screen and (max-width:772px){body{background:none;margin-top:0;max-width:none;padding-right:0}}#logo{background:url(//www.google.com/images/branding/googlelogo/1x/googlelogo_color_150x54dp.png) no-repeat;margin-left:-5px}@media only screen and (min-resolution:192dpi){#logo{background:url(//www.google.com/images/branding/googlelogo/2x/googlelogo_color_150x54dp.png) no-repeat 0% 0%/100% 100%;-moz-border-image:url(//www.google.com/images/branding/googlelogo/2x/googlelogo_color_150x54dp.png) 0}}@media only screen and (-webkit-min-device-pixel-ratio:2){#logo{background:url(//www.google.com/images/branding/googlelogo/2x/googlelogo_color_150x54dp.png) no-repeat;-webkit-background-size:100% 100%}}#logo{display:inline-block;height:54px;width:150px}
//│   </style>
//│   <a href=//www.google.com/><span id=logo aria-label=Google></span></a>
//│   <p><b>404.</b> <ins>That’s an error.</ins>
//│   <p>The requested URL <code>/v1/alloy-adhoc-hessjc/instances?alt=json&amp;instanceId=dbinstance</code> was not found on this server.  <ins>That’s all we know.</ins>
//│
//│
//│   with google_alloydb_instance.private_alloydb,
//│   on private-database.tf line 95, in resource "google_alloydb_instance" "private_alloydb":
//│   95: resource "google_alloydb_instance" "private_alloydb" {

#resource "google_alloydb_instance" "private_alloydb" {
#  cluster       = google_alloydb_cluster.private_alloydb.cluster_id
#  instance_type = "PRIMARY"
#  instance_id   = local.alloydb_instance_id
#
#  machine_config {
#    cpu_count = 2
#  }
#
#  depends_on = [google_service_networking_connection.private_vpc_connection, google_alloydb_cluster.private_alloydb]
#}

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

output "alloy_db_instance_uri" {
  value = local.alloydb_instance_uri
}
