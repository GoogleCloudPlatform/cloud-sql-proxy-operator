# Quick Start

Follow the instructions in the Quick Start Guide for Cloud SQL: 
[Connect to Cloud SQL for PostgreSQL from Google Kubernetes Engine](
https://cloud.google.com/sql/docs/postgres/connect-instance-kubernetes)
through the end of the step named [Build the Sample App](
https://cloud.google.com/sql/docs/postgres/connect-instance-kubernetes#build_the_sample_app).

Then, continue following these instructions:

## Install the Cloud SQL Proxy Operator

Confirm that kubectl can connect to the cluster.

```shell
kubectl cluster-info
```

Run the following command to install the cloud sql proxy operator into
your kubernetes cluster:

<!-- {x-release-please-start-version} -->
```shell
curl https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy-operator/v1.5.0/install.sh | bash
```
<!-- {x-release-please-end} -->

This will use `helm` to install the `cert-manager` operator, a prerequisite. Then
it will install the Cloud SQL Proxy Operator in your cluster.

Wait for the Cloud SQL Auth Proxy Operator to start.

```shell
kubectl rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s
```

Confirm that the operator is installed and running by listing its pods:

```shell
kubectl get pods -n cloud-sql-proxy-operator-system
```

## Configure Cloud SQL Proxy for the quick start app

Get the Cloud SQL instance connection name by running the gcloud sql instances describe command:

```shell
gcloud sql instances describe quickstart-instance --format='value(connectionName)'
```

Create a new file named `authproxyworkload.yaml` containing the following:

```yaml
apiVersion: cloudsql.cloud.google.com/v1
kind: AuthProxyWorkload
metadata:
  name: authproxyworkload-sample
spec:
  workloadSelector:
    kind: "Deployment"
    name: "gke-cloud-sql-quickstart"
  instances:
  - connectionString: "<INSTANCE_CONNECTION_NAME>"
    portEnvName: "DB_PORT"
    hostEnvName: "INSTANCE_HOST"
```

Update `<INSTANCE_CONNECTION_NAME>` with the Cloud SQL instance connection name
retrieved from the gcloud command on the previous step. This should follow the format
project_id:region:instance_name. The instance connection name is also visible
in the Google Cloud Console on the Cloud SQL Instance Overview page.

Apply the proxy configuration to kubernetes:

```shell
kubectl apply -f authproxyworkload.yaml
```

### Deploy the sample app

Proceed with the quickstart guide step [Deploy the sample app](
https://cloud.google.com/sql/docs/postgres/connect-instance-kubernetes#deploy_the_sample_app).
In step 2, use this YAML as your template.

Note that this template has only one container for the application. In the published
quickstart guide, there are two containers, one for the application, and one for the
proxy.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gke-cloud-sql-quickstart
spec:
  selector:
    matchLabels:
      app: gke-cloud-sql-app
  template:
    metadata:
      labels:
        app: gke-cloud-sql-app
    spec:
      serviceAccountName: <YOUR-KSA-NAME>
      containers:
      - name: gke-cloud-sql-app
        # Replace <LOCATION> with your Artifact Registry location (e.g., us-central1).
        # Replace <YOUR_PROJECT_ID> with your project ID.
        image: <LOCATION>-docker.pkg.dev/<YOUR_PROJECT_ID>/gke-cloud-sql-repo/gke-sql:latest
        # This app listens on port 8080 for web traffic by default.
        ports:
        - containerPort: 8080
        env:
        - name: PORT
          value: "8080"
        - name: INSTANCE_HOST
          value: "set-by-proxy"
        - name: DB_PORT
          value: "set-by-proxy"
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: <YOUR-DB-SECRET>
              key: username
        - name: DB_PASS
          valueFrom:
            secretKeyRef:
              name: <YOUR-DB-SECRET>
              key: password
        - name: DB_NAME
          valueFrom:
            secretKeyRef:
              name: <YOUR-DB-SECRET>
              key: database
```

### Inspect the container managed by the proxy operator
Finally, after completing the steps in the quickstart guide, inspect the pods
for the application to see the proxy container.

```shell
kubectl describe pods -l app=gke-cloud-sql-app
```

Note that there are now two containers in the pods, while there is only one
container in the deployment. The operator adds a second proxy container configured
using the settings in the `AuthProxyWorkload` resource. 
