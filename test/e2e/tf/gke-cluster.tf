# From https://github.com/hashicorp/terraform-provider-kubernetes/blob/main/kubernetes/test-infra/gke/main.tf

variable "kubernetes_version" {
  default = ""
}

variable "workers_count" {
  default = "2"
}

variable "node_machine_type" {
  default = "e2-standard-2"
}

variable "enable_alpha" {
  default = false
}

data "google_client_config" "default" {
}

data "google_container_engine_versions" "supported" {
  location       = local.gcloud_zone
  version_prefix = var.kubernetes_version
  project = var.gcloud_project_id
}

resource "random_id" "cluster_name" {
  byte_length = 10
}

resource "google_service_account" "node_pool" {
  account_id   = "k8s-nodes-${random_id.cluster_name.hex}"
  display_name = "Kubernetes provider SA"
  project = var.gcloud_project_id
}

resource "google_project_iam_binding" "cloudsql_client" {
  project = var.gcloud_project_id
  role               = "roles/cloudsql.client"
  members = [
    "serviceAccount:${google_service_account.node_pool.email}"
  ]
}
resource "google_project_iam_member" "allow_image_pull" {
  project = var.gcloud_project_id
  role   = "roles/artifactregistry.reader"
  member = "serviceAccount:${google_service_account.node_pool.email}"
}

resource "google_container_cluster" "primary" {
  project = var.gcloud_project_id
  name               = "operator-test-${random_id.cluster_name.hex}"
  location           = local.gcloud_zone
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

}

resource "google_container_node_pool" "primary_preemptible_nodes" {
  name       = "operator-test-nodes-${random_id.cluster_name.hex}"
  cluster    = google_container_cluster.primary.id
  initial_node_count = var.workers_count
  version =  data.google_container_engine_versions.supported.latest_node_version
  location = local.gcloud_zone

  autoscaling {
    max_node_count = 10
    min_node_count = 2
  }

  management {
    auto_repair  = var.enable_alpha ? false : true
    auto_upgrade = var.enable_alpha ? false : true
  }

  node_config {
    preemptible  = true
    machine_type = "e2-standard-8"

    # Google recommends custom service accounts that have cloud-platform scope and permissions granted via IAM Roles.
    service_account = google_service_account.node_pool.email
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
  kubeconfig = {
    apiVersion = "v1"
    kind       = "Config"
    preferences = {
      colors = true
    }
    current-context = google_container_cluster.primary.name
    contexts = [
      {
        name = google_container_cluster.primary.name
        context = {
          cluster   = google_container_cluster.primary.name
          user      = google_service_account.node_pool.email
          namespace = "default"
        }
      }
    ]
    clusters = [
      {
        name = google_container_cluster.primary.name
        cluster = {
          server                     = "https://${google_container_cluster.primary.endpoint}"
          certificate-authority-data = google_container_cluster.primary.master_auth[0].cluster_ca_certificate
        }
      }
    ]
    users = [
      {
        name = google_service_account.node_pool.email
        user = {
          exec = {
            apiVersion= "client.authentication.k8s.io/v1beta1"
            command = "gke-gcloud-auth-plugin"
            installHint =  "Install gke-gcloud-auth-plugin for use with kubectl by following https://cloud.google.com/blog/products/containers-kubernetes/kubectl-auth-changes-in-gke"
            provideClusterInfo = true
          }
# Disable auth section, this
#          auth-provider = {
#            name = "gcp"
#            config = {
#              cmd-args = "config config-helper --format=json"
#              cmd-path = var.gcloud_bin
#              expiry-key = "{.credential.token_expiry}"
#              token-key = "{.credential.access_token}"
#            }
#          }
        }
      }
    ]
  }

}

resource "local_file" "kubeconfig" {
  content  = yamlencode(local.kubeconfig)
  filename = var.kubeconfig_path
}

#resource "local_file" "gcptoken" {
#  content  = data.google_client_config.default.access_token
#  filename = "${path.module}/gcptoken"
#}

#output "google_zone" {
#  value = local.gcloud_zone
#}

output "node_version" {
  value = google_container_node_pool.primary_preemptible_nodes.version
}

output "kubeconfig_path" {
  value = local_file.kubeconfig.filename
}

output "cluster_name" {
  value = google_container_cluster.primary.name
}