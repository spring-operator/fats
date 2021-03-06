#!/bin/bash

echo -e "${ANSI_RED}NOTE: ECR will not fully work until https://github.com/knative/serving/issues/1996 is resolved${ANSI_RESET}"

# Install aws for ECR access
`dirname "${BASH_SOURCE[0]}"`/../../install.sh aws

# Login for local pushes
$(aws ecr get-login --no-include-email --region us-west-2)

IMAGE_REPOSITORY_PREFIX="$(aws sts get-caller-identity --output text --query 'Account').dkr.ecr.us-west-2.amazonaws.com"
NAMESPACE_INIT_FLAGS="${NAMESPACE_INIT_FLAGS:-} --secret push-credentials"

fats_image_repo() {
  local function_name=$1

  # ECR requires the repo be created before pushing an image.
  # allow to fail since the repository may already exist
  aws ecr create-repository --repository-name $function_name --region us-west-2 1>&2 || true

  echo -n "${IMAGE_REPOSITORY_PREFIX}/${function_name}:${CLUSTER_NAME}"
}

fats_delete_image() {
  local image=$1
  IFS=':' read -r -a image <<< "$1"
  local repo=${image[0]}
  local tag=${image[1]}

  aws ecr batch-delete-image --repository-name $repo --image-ids imageTag=$tag
}

fats_create_push_credentials() {
  local namespace=$1

  local token=`aws ecr get-authorization-token --region us-west-2 --output text --query 'authorizationData[].authorizationToken' | base64 --decode`
  local username=`echo $token | cut -d':' -f1`
  local password=`echo $token | cut -d':' -f2`
  local endpoint="https://$(aws sts get-caller-identity --output text --query 'Account').dkr.ecr.us-west-2.amazonaws.com/v2/"

  echo "Create auth secret"
  cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: push-credentials
  namespace: $(echo -n "$namespace")
  annotations:
    build.knative.dev/docker-0: $(echo -n "$endpoint")
type: kubernetes.io/basic-auth
data:
  username: $(echo -n "$username" | openssl base64 -a -A)
  password: $(echo -n "$password" | openssl base64 -a -A)
EOF
}
