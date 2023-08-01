##
# Create a subnet and serverless VPC connector for cloud run instances
#
# See https://cloud.google.com/run/docs/configuring/connecting-vpc#terraform

resource "google_compute_subnetwork" "cloud_run_network" {
  provider = google-beta
  project  = var.project_id
  region   = var.gcloud_region

  name          = "test-vpc-cloudrun-${var.environment_name}"
  ip_cidr_range = "10.3.1.0/28"
  network       = google_compute_network.private_k8s_network.id

}


# It appears that the google experimental projects can't do cloud run vpc connectors
# This error shows up in the audit log when terraform tries to create the
# serverless-connector module:
#
#    Constraint constraints/compute.trustedImageProjects violated for project
#    hessjc-csql-operator-02. Use of images from project serverless-vpc-access-images
#    is prohibited.

#
#module "serverless-connector" {
#  source     = "terraform-google-modules/network/google//modules/vpc-serverless-connector-beta"
#  version    = "~> 7.0"
#  project_id = var.project_id
#  vpc_connectors = [{
#    name        = "centralcloudrun"
#    region      = var.gcloud_region
#    subnet_name = google_compute_subnetwork.cloud_run_network.name
#    machine_type  = "e2-micro"
#    min_instances = 2
#    max_instances = 7
#  }]
#}
