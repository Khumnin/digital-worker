#!/usr/bin/env bash
# eks-setup.sh — One-time EKS provisioning + first deploy for tigersoft-auth
# Run from anywhere; paths are resolved relative to this script.
# Requirements: aws CLI, kubectl (configured for the cluster), docker, openssl
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/../backend"
K8S_DIR="$SCRIPT_DIR/../k8s"
REGION="ap-southeast-7"
APP="tigersoft-auth"
NAMESPACE="henderson"
DB_NS="database"
DB_SVC="central-postgres"
DB_NAME="auth_system"
DB_USER="auth_user"
KONG_ADMIN="http://kong-kong-admin.kong.svc.cluster.local:8001"

echo "==> tigersoft-auth EKS Setup"
echo "    Region   : $REGION"
echo "    Namespace: $NAMESPACE"
echo ""

# ── 0. Verify prerequisites ───────────────────────────────────────────────────
for cmd in aws kubectl docker openssl; do
  if ! command -v "$cmd" &>/dev/null; then
    echo "ERROR: '$cmd' not found. Please install it first."
    exit 1
  fi
done

# ── 1. Detect AWS Account ID ──────────────────────────────────────────────────
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_REGISTRY="${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com"
IMAGE="${ECR_REGISTRY}/${APP}"
echo "[1/9] AWS Account: $ACCOUNT_ID"
echo "      ECR Registry: $ECR_REGISTRY"

# ── 2. Create ECR repository (idempotent) ────────────────────────────────────
if aws ecr describe-repositories --repository-names "$APP" --region "$REGION" &>/dev/null; then
  echo "[2/9] ECR repository '$APP' already exists"
else
  aws ecr create-repository \
    --repository-name "$APP" \
    --region "$REGION" \
    --image-scanning-configuration scanOnPush=true \
    --image-tag-mutability MUTABLE
  echo "[2/9] ECR repository '$APP' created"
fi

# ── 3. Create PostgreSQL database + user ─────────────────────────────────────
echo "[3/9] Setting up PostgreSQL database..."
DB_PASSWORD=$(openssl rand -hex 24)

# Find postgres pod
PG_POD=$(kubectl get pod -n "$DB_NS" -l app=postgres \
  --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || \
  kubectl get pod -n "$DB_NS" --field-selector=status.phase=Running \
  -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

if [ -z "$PG_POD" ]; then
  echo "WARN: Could not find a running postgres pod in namespace '$DB_NS'."
  echo "      Set DATABASE_URL manually and skip this step."
  DB_PASSWORD="REPLACE_ME_WITH_DB_PASSWORD"
else
  kubectl exec -n "$DB_NS" "$PG_POD" -- psql -U postgres -c \
    "CREATE USER ${DB_USER} WITH PASSWORD '${DB_PASSWORD}';" 2>/dev/null || \
    echo "      User '${DB_USER}' already exists — skipping"

  kubectl exec -n "$DB_NS" "$PG_POD" -- psql -U postgres -c \
    "CREATE DATABASE ${DB_NAME} OWNER ${DB_USER};" 2>/dev/null || \
    echo "      Database '${DB_NAME}' already exists — skipping"

  echo "      Database and user ready"
fi

DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_SVC}.${DB_NS}.svc.cluster.local:5432/${DB_NAME}?sslmode=disable"

# ── 4. Generate RSA-2048 signing key ─────────────────────────────────────────
KEY_DIR=$(mktemp -d)
trap 'rm -rf "$KEY_DIR"' EXIT
openssl genrsa -out "$KEY_DIR/private.pem" 2048 2>/dev/null
JWT_PEM=$(cat "$KEY_DIR/private.pem")
echo "[4/9] RSA-2048 signing key generated"

# ── 5. Collect user input ─────────────────────────────────────────────────────
echo ""
read -rp "[5/9] Your domain (e.g. yourdomain.com): " DOMAIN
read -rp "      Resend API key (re_...): " RESEND_KEY
read -rp "      FROM email address: " EMAIL_FROM
echo ""

APP_URL="https://auth.${DOMAIN}"

# ── 6. Create Kubernetes namespace + secret ───────────────────────────────────
echo "[6/9] Creating namespace '$NAMESPACE'..."
kubectl apply -f "$K8S_DIR/namespace.yaml"

echo "      Creating Kubernetes secret..."
kubectl create secret generic auth-api-secrets \
  --namespace="$NAMESPACE" \
  --from-literal="DATABASE_URL=$DATABASE_URL" \
  --from-literal="REDIS_URL=redis://my-redis-master.redis.svc.cluster.local:6379" \
  --from-literal="JWT_RSA_PRIVATE_KEY_PEM=$JWT_PEM" \
  --from-literal="RESEND_API_KEY=$RESEND_KEY" \
  --from-literal="OAUTH_GOOGLE_CLIENT_SECRET=" \
  --dry-run=client -o yaml | kubectl apply -f -

# Update ConfigMap with actual domain
sed "s|https://auth.yourdomain.com|${APP_URL}|g; s|auth@yourdomain.com|${EMAIL_FROM}|g" \
  "$K8S_DIR/configmap.yaml" | kubectl apply -f -

echo "      Secrets and config applied"

# ── 7. Build + push Docker image ─────────────────────────────────────────────
echo "[7/9] Building and pushing Docker image..."
aws ecr get-login-password --region "$REGION" | \
  docker login --username AWS --password-stdin "$ECR_REGISTRY"

GIT_SHA=$(git -C "$BACKEND_DIR" rev-parse --short HEAD 2>/dev/null || echo "latest")
FULL_IMAGE="${IMAGE}:${GIT_SHA}"

docker build -t "$FULL_IMAGE" "$BACKEND_DIR"
docker push "$FULL_IMAGE"
docker tag "$FULL_IMAGE" "${IMAGE}:latest"
docker push "${IMAGE}:latest"
echo "      Image pushed: $FULL_IMAGE"

# ── 8. Deploy to EKS ─────────────────────────────────────────────────────────
echo "[8/9] Deploying to EKS..."

# Patch the image placeholder in deployment.yaml before applying
sed "s|REGISTRY_PLACEHOLDER/tigersoft-auth:latest|${FULL_IMAGE}|g" \
  "$K8S_DIR/deployment.yaml" | kubectl apply -f -

# Apply remaining manifests (namespace already applied, configmap already applied)
kubectl apply -f "$K8S_DIR/serviceaccount.yaml"
kubectl apply -f "$K8S_DIR/service.yaml"
kubectl apply -f "$K8S_DIR/hpa.yaml"

kubectl rollout status deployment/auth-api -n "$NAMESPACE" --timeout=300s
echo "      Deployment complete"

# ── 9. Register with Kong API Gateway ────────────────────────────────────────
echo "[9/9] Registering with Kong..."

# Run registration from a temporary pod inside the cluster
kubectl run kong-reg --rm -i --restart=Never \
  --image=curlimages/curl:latest \
  --namespace=kong \
  -- sh -c "
    # Create upstream
    curl -sf -X PUT ${KONG_ADMIN}/upstreams/auth-system \
      -d 'name=auth-system' || true

    # Add target (auth service in henderson namespace)
    curl -sf -X POST ${KONG_ADMIN}/upstreams/auth-system/targets \
      -d 'target=auth-system.henderson.svc.cluster.local:8080' || true

    # Create Kong service
    curl -sf -X PUT ${KONG_ADMIN}/services/auth-system \
      -d 'name=auth-system' \
      -d 'host=auth-system' \
      -d 'port=8080' \
      -d 'protocol=http' || true

    # Create Kong route (host-based)
    curl -sf -X POST ${KONG_ADMIN}/services/auth-system/routes \
      -d 'name=auth-system-route' \
      -d 'hosts[]=${APP_URL#https://}' \
      -d 'strip_path=false' || true

    echo 'Kong registration done'
  " 2>/dev/null && echo "      Kong registration successful" || \
    echo "WARN: Kong registration failed — add the service manually via Konga"

echo ""
echo "✓ Setup complete!"
echo ""
echo "  App URL  : $APP_URL"
echo "  Health   : Run: kubectl port-forward svc/auth-system 8080:8080 -n $NAMESPACE"
echo "             Then: curl http://localhost:8080/health"
echo ""
echo "Next steps:"
echo "  1. Add DNS: auth.${DOMAIN} → Cloudflare Tunnel (via Zero Trust dashboard)"
echo "     Set the tunnel backend to: http://kong-kong-proxy.kong.svc.cluster.local:80"
echo "     with custom header: Host: auth.${DOMAIN}"
echo ""
echo "  2. Add GitHub repo secrets for CI/CD auto-deploy:"
echo "     AWS_ROLE_ARN  = <IAM role with ECR push + EKS deploy>"
echo "     ECR_REGISTRY  = ${ECR_REGISTRY}"
echo "     EKS_CLUSTER_NAME = <your cluster name>"
