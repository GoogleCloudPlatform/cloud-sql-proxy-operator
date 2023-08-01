#!/bin/bash
set -euxo

########
### Cloud build following the Cloud Run example

#IMAGE="us-central1-docker.pkg.dev/hessjc-csql-operator-02/testhessjc/e2e-java-cloudrun"
#gcloud builds submit --project=hessjc-csql-operator-02 --pack

#gcloud run jobs create job-quickstart \
#    --image $IMAGE \
#    --tasks 50 \
#    --set-env-vars SLEEP_MS=10000 \
#    --set-env-vars FAIL_RATE=0.5 \
#    --max-retries 5 \
#    --region us-central1 \
#    --project=hessjc-csql-operator-02
#
#gcloud run jobs execute job-quickstart \
#    --region us-central1 \
#    --project=hessjc-csql-operator-02


function build_image() {
  ########
  ### Local build with Jib pushing to the project's container repo
  mvn install
}

function run_job() {
  DB_TYPE=$1
  CONNECT=$2
  INST=$3
  DB_USER=$4
  IP_TYPES=$5

  JOB_NAME=job-e2e-java-$DB_TYPE-$CONNECT
  IMAGE=$(jq -r '(.image + "@"+ .imageDigest)' < target/jib-image.json )

  if [[ "$IP_TYPES" == "PRIVATE" ]] ; then
    private_args=("--project=hessjc-csql-operator-02" --vpc-connector "centralcloudrun" --set-env-vars "E2E_IP_TYPES=$IP_TYPES" )
    JOB_NAME="$JOB_NAME-private"
  else
    private_args=("--project=hessjc-csql-operator-02")
  fi

  if gcloud run jobs describe "$JOB_NAME" --region us-central1 --project=hessjc-csql-operator-02 > /dev/null 2>&1 ; then
    gcloud run jobs update "$JOB_NAME" --region us-central1 --project=hessjc-csql-operator-02 --image "$IMAGE"
  else
    gcloud beta run jobs create "$JOB_NAME" \
      --image "$IMAGE" \
      --tasks 2 \
      --task-timeout 2h30m \
      --set-env-vars DB_NAME=db \
      --set-env-vars DB_NAME_1=db1 \
      --set-env-vars "DB_USER=$DB_USER" \
      --set-env-vars DB_PASS=604a0cc12f342b9ae9f9 \
      --set-env-vars "DB_INSTANCE=$INST" \
      --set-env-vars "E2E_CONNECT_PATTERN=$CONNECT" \
      --set-env-vars "E2E_DB_TYPE=$DB_TYPE" \
      --max-retries 2 \
      --region us-central1 \
      "${private_args[@]}"
  fi
  gcloud beta run jobs execute "$JOB_NAME" \
      --region us-central1 \
      --project=hessjc-csql-operator-02
}

build_image
run_job mysql short "hessjc-csql-operator-02:us-central1:mysql2d971003318a3077402fhessjc" "dbuser" "PUBLIC"
run_job mysql long "hessjc-csql-operator-02:us-central1:mysql2d971003318a3077402fhessjc" "dbuser" "PUBLIC"
run_job postgres long "hessjc-csql-operator-02:us-central1:inst2d971003318a3077402fhessjc" "postgres" "PUBLIC"
run_job postgres short "hessjc-csql-operator-02:us-central1:inst2d971003318a3077402fhessjc" "postgres" "PUBLIC"

# Currently serverless vpc connect is not working
#run_job postgres long "hessjc-csql-operator-02:us-central1:privateinstbbf4eae6hessjc" "postgres" "PRIVATE"
#run_job postgres short "hessjc-csql-operator-02:us-central1:privateinstbbf4eae6hessjc" "postgres" "PRIVATE"

