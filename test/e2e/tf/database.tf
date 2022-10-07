resource "random_id" "db_name" {
  byte_length = 10
}

resource "random_id" "db_password" {
  byte_length = 10
}

# See versions at https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/sql_database_instance#database_version
resource "google_sql_database_instance" "instance" {
  name             = "inst${random_id.db_name.hex}"
  project          = var.project_id
  region           = var.gcloud_region
  database_version = "POSTGRES_13"
  settings {
    tier = "db-f1-micro"
  }
  deletion_protection = "true"
  root_password       = random_id.db_password.hex
}

resource "google_sql_database" "db" {
  name     = "db"
  instance = google_sql_database_instance.instance.name
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
