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
      version = "4.48.0"
    }
  }
}


provider "google" {
}

# Enable gcloud project APIs
locals {
  standard_labels = {
    e2e_test_resource = "true"
    landscape         = var.environment_name
  }
}

##
# This is how you do an output file containing terraform data for use by
# a subsequent script.

# First, create the output data structure as a local variable
locals {
  output_json = {
    instance     = google_sql_database_instance.instance.connection_name
    db           = google_sql_database.db.name
    rootPassword = random_id.db_password.hex
    kubeconfig   = var.kubeconfig_path
  }
}

# Then write the output data to a local file in json format
resource "local_file" "testinfra" {
  content  = jsonencode(local.output_json)
  filename = var.output_json_path
}
