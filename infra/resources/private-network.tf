
resource "google_compute_network" "private_k8s_network" {
  provider = google-beta
  project = var.project_id

  name                    = "test-network"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "private_k8s_network" {
  provider = google-beta
  project = var.project_id
  region = var.gcloud_region

  name          = "test-subnetwork"
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
}

resource "google_compute_global_address" "private_ip_address" {
  provider = google-beta
  project = var.project_id

  name          = "private-ip-address"
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
