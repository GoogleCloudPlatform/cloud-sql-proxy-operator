# Copyright 2023 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


resource "google_compute_network" "private_k8s_network" {
  provider = google-beta
  project  = var.project_id

  name                    = "test-vpc-${var.environment_name}"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "private_k8s_network" {
  provider = google-beta
  project  = var.project_id
  region   = var.gcloud_region

  name          = "test-vpc-subnetwork-${var.environment_name}"
  ip_cidr_range = "10.2.0.0/16"
  network       = google_compute_network.private_k8s_network.id

  secondary_ip_range {
    range_name    = "services-range"
    ip_cidr_range = "192.168.1.0/24"
  }

  secondary_ip_range {
    range_name    = "pod-ranges"
    ip_cidr_range = "192.168.64.0/22"
  }

  secondary_ip_range {
    range_name    = "nodepool-pod-ranges"
    ip_cidr_range = "192.168.128.0/22"
  }
}

resource "google_compute_global_address" "private_ip_address" {
  provider = google-beta
  project  = var.project_id

  name          = "test-vpc-private-ip-address-${var.environment_name}"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = google_compute_network.private_k8s_network.id
}

resource "google_service_networking_connection" "private_vpc_connection" {
  provider = google-beta

  network                 = google_compute_network.private_k8s_network.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_address.name]
}
