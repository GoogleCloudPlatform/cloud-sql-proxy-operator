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

resource "random_id" "db_name" {
  byte_length = 10
}

resource "random_id" "db_password" {
  byte_length = 10
}

# See versions at https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/sql_database_instance#database_version
resource "google_sql_database_instance" "instance" {
  name             = "inst${random_id.db_name.hex}${var.environment_name}"
  project          = var.project_id
  region           = var.gcloud_region
  database_version = "POSTGRES_13"
  settings {
    tier        = "db-f1-micro"
    user_labels = local.standard_labels
  }
  deletion_protection = "true"
  root_password       = random_id.db_password.hex
}

resource "google_sql_database" "db" {
  name     = "db"
  instance = google_sql_database_instance.instance.name
  project  = var.project_id
}
resource "google_sql_database" "db1" {
  name     = "db1"
  instance = google_sql_database_instance.instance.name
  project  = var.project_id
}

# See versions at https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/sql_database_instance#database_version
resource "google_sql_database_instance" "mysql" {
  name             = "mysql${random_id.db_name.hex}${var.environment_name}"
  project          = var.project_id
  region           = var.gcloud_region
  database_version = "MYSQL_8_0"
  settings {
    tier        = "db-f1-micro"
    user_labels = local.standard_labels
  }
  deletion_protection = "true"
  root_password       = random_id.db_password.hex
}

resource "google_sql_database" "db_mysql" {
  name     = "db"
  instance = google_sql_database_instance.mysql.name
  project  = var.project_id
}
resource "google_sql_database" "db_mysql_1" {
  name     = "db1"
  instance = google_sql_database_instance.mysql.name
  project  = var.project_id
}

resource "google_sql_user" "mysql_user" {
  name     = "dbuser"
  instance = google_sql_database_instance.mysql.name
  host     = "%"
  password = random_id.db_password.hex
  project  = var.project_id
}

# See versions at https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/sql_database_instance#database_version
resource "google_sql_database_instance" "mssql" {
  name             = "mssql${random_id.db_name.hex}${var.environment_name}"
  project          = var.project_id
  region           = var.gcloud_region
  database_version = "SQLSERVER_2019_EXPRESS"
  settings {
    # SQL Server tier format: "db-custom-$CPU-$MEM" where $CPU is the number of
    # cores and $MEM is memory in MiB
    tier        = "db-custom-2-3840"
    user_labels = local.standard_labels
  }
  deletion_protection = "true"
  root_password       = random_id.db_password.hex
}

resource "google_sql_database" "db_mssql" {
  name     = "db"
  instance = google_sql_database_instance.mssql.name
  project  = var.project_id
}

resource "google_sql_database" "db_mssql_1" {
  name     = "db1"
  instance = google_sql_database_instance.mssql.name
  project  = var.project_id
}

resource "google_sql_user" "mssql_user" {
  name     = "dbuser"
  instance = google_sql_database_instance.mssql.name
  password = random_id.db_password.hex
  project  = var.project_id
}

output "db_root_password" {
  value = random_id.db_password.hex
}
output "db_instance_name" {
  value = google_sql_database_instance.instance.name
}
output "db_database_name" {
  value = google_sql_database.db.name
}
