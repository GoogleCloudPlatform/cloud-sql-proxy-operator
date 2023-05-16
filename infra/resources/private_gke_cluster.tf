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

# From https://github.com/hashicorp/terraform-provider-kubernetes/blob/main/kubernetes/test-infra/gke/main.tf

resource "google_container_cluster" "private" {
  project            = var.project_id
  name               = "operator-private-${var.environment_name}"
  location           = var.gcloud_zone
  min_master_version = data.google_container_engine_versions.supported.latest_master_version
  initial_node_count = 2

  // Alpha features are disabled by default and can be enabled by GKE for a particular GKE control plane version.
  // Creating an alpha cluster enables all alpha features by default.
  // Ref: https://cloud.google.com/kubernetes-engine/docs/concepts/feature-gates
  enable_kubernetes_alpha = var.enable_alpha

  // disalbe the default nodepool and specify node pools as
  // separate terraform resources. This way if we
  // change the nodepool config, we don't delete the cluster too
  remove_default_node_pool = true
  resource_labels          = local.standard_labels

  network    = google_compute_network.private_k8s_network.id
  subnetwork = google_compute_subnetwork.private_k8s_network.id

  ip_allocation_policy {
    services_secondary_range_name = google_compute_subnetwork.private_k8s_network.secondary_ip_range.0.range_name
    cluster_secondary_range_name  = google_compute_subnetwork.private_k8s_network.secondary_ip_range.1.range_name
  }
}

resource "google_container_node_pool" "private_preemptible_nodes" {
  name               = "operator-private-nodes-${var.environment_name}"
  cluster            = google_container_cluster.private.id
  initial_node_count = var.workers_count
  version            = data.google_container_engine_versions.supported.latest_master_version
  location           = var.gcloud_zone

  autoscaling {
    max_node_count = 10
    min_node_count = 2
  }

  management {
    auto_repair  = var.enable_alpha ? false : true
    auto_upgrade = var.enable_alpha ? false : true
  }

  network_config {
    enable_private_nodes = false
    pod_range            = google_compute_subnetwork.private_k8s_network.secondary_ip_range.2.range_name
    create_pod_range     = false
  }

  node_config {
    preemptible     = true
    machine_type    = "e2-standard-8"
    resource_labels = local.standard_labels

    # Google recommends custom service accounts that have cloud-platform scope and permissions granted via IAM Roles.
    service_account = var.nodepool_serviceaccount_email
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform",
      "https://www.googleapis.com/auth/compute",
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
      "https://www.googleapis.com/auth/sqlservice.admin",
    ]
  }
}


locals {
  # This is the recommended way to produce a kubeconfig file from
  # the Google Cloud GKE terraform resource.
  private_kubeconfig = {
    apiVersion = "v1"
    kind       = "Config"
    preferences = {
      colors = true
    }
    current-context = google_container_cluster.private.name
    contexts = [
      {
        name = google_container_cluster.private.name
        context = {
          cluster   = google_container_cluster.private.name
          user      = var.nodepool_serviceaccount_email
          namespace = "default"
        }
      }
    ]
    clusters = [
      {
        name = google_container_cluster.private.name
        cluster = {
          server                     = "https://${google_container_cluster.private.endpoint}"
          certificate-authority-data = google_container_cluster.private.master_auth[0].cluster_ca_certificate
        }
      }
    ]
    users = [
      {
        name = var.nodepool_serviceaccount_email
        user = {
          exec = {
            apiVersion         = "client.authentication.k8s.io/v1beta1"
            command            = "gke-gcloud-auth-plugin"
            installHint        = "Install gke-gcloud-auth-plugin for use with kubectl by following https://cloud.google.com/blog/products/containers-kubernetes/kubectl-auth-changes-in-gke"
            provideClusterInfo = true
          }
        }
      }
    ]
  }

}

resource "local_file" "private_kubeconfig" {
  content  = yamlencode(local.private_kubeconfig)
  filename = var.private_kubeconfig_path
  file_permission = "0600"
}
