# SealedSecrets Setup Guide

This document explains how to use SealedSecrets to securely manage secrets in your Kubernetes cluster.

## Overview

SealedSecrets is a Kubernetes controller that encrypts secrets so they can be safely committed to Git. The controller runs in your cluster and decrypts the sealed secrets into regular Kubernetes secrets.

## Prerequisites

- kubectl configured with cluster access
- kubeseal CLI tool installed
- Helm installed (for the SealedSecrets controller)

## Installation

### 1. Install kubeseal CLI

```bash
# Linux
wget https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.24.0/kubeseal-linux-amd64 -O kubeseal
sudo install -m 755 kubeseal /usr/local/bin/kubeseal

# macOS
brew install kubeseal
```

### 2. Deploy SealedSecrets Controller

The SealedSecrets controller is deployed via Helmfile:

```bash
cd helm/infra
helmfile apply
```

Or deploy it manually:

```bash
kubectl apply -f k8s/base/sealed-secrets-controller.yaml
```

### 3. Verify Installation

```bash
kubectl get pods -n kube-system | grep sealed-secrets
kubectl get deployment -n kube-system sealed-secrets-controller
```

## Creating SealedSecrets

### For Microservices (K8s manifests)

#### Step 1: Create a regular secret template

Edit the secret-template.yaml file in each service directory:

```bash
# Example: k8s/base/analytics/secret-template.yaml
apiVersion: v1
kind: Secret
metadata:
  name: analytics-secret
  namespace: url-shortener
type: Opaque
stringData:
  DATABASE_URL: "postgresql://user:password@host:port/database"
  KAFKA_BROKERS: "kafka:9092"
  API_KEY: "your-api-key"
```

#### Step 2: Encrypt the secret

```bash
# From the project root
kubectl create secret generic analytics-secret \
  --namespace=url-shortener \
  --from-literal=DATABASE_URL="postgresql://user:password@host:port/database" \
  --from-literal=KAFKA_BROKERS="kafka:9092" \
  --from-literal=API_KEY="your-api-key" \
  --dry-run=client -o yaml | kubeseal --format yaml > k8s/base/analytics/sealed-secret.yaml
```

#### Step 3: Update kustomization.yaml

Uncomment the sealed-secret line in `k8s/base/kustomization.yaml`:

```yaml
resources:
  # - gateway/secret.yaml  # Replaced with SealedSecret
  - gateway/sealed-secret.yaml  # Add after encrypting with kubeseal
```

#### Step 4: Apply the changes

```bash
kubectl apply -k k8s/base
```

### For Helm Charts (Infrastructure)

#### Step 1: Create a regular secret

```bash
# PostgreSQL
kubectl create secret generic postgres-secret \
  --namespace=url-shortener \
  --from-literal=postgres-password="your-postgres-password" \
  --from-literal=password="your-user-password" \
  --dry-run=client -o yaml | kubeseal --format yaml > helm/infra/postgres/sealed-secret.yaml

# Redis
kubectl create secret generic redis-secret \
  --namespace=url-shortener \
  --from-literal=password="your-redis-password" \
  --from-literal=app-password="your-app-password" \
  --dry-run=client -o yaml | kubeseal --format yaml > helm/infra/redis/sealed-secret.yaml

# MongoDB
kubectl create secret generic mongo-secret \
  --namespace=url-shortener \
  --from-literal=root-password="your-mongo-password" \
  --dry-run=client -o yaml | kubeseal --format yaml > helm/infra/mongo/sealed-secret.yaml
```

#### Step 2: Update Helmfile to use secrets

Modify `helm/infra/helmfile.yaml` to reference the sealed secrets:

```yaml
releases:
  - name: postgres
    namespace: url-shortener
    chart: bitnami/postgresql
    values:
      - ./postgres/values.yaml
    set:
      - name: auth.postgresPassword
        value: "{{ .Values.postgresPassword }}"
      - name: auth.password
        value: "{{ .Values.password }}"
```

Or use a values override file that references the secret:

```yaml
# helm/infra/postgres/values-override.yaml
auth:
  postgresPassword: ""
  password: ""
existingSecret: postgres-secret
```

#### Step 3: Apply the changes

```bash
cd helm/infra
helmfile apply
```

## Managing SealedSecrets

### Updating a Secret

1. Update the secret values in the sealed-secret.yaml file
2. Re-encrypt using kubeseal
3. Apply the updated sealed-secret

```bash
kubectl apply -f k8s/base/analytics/sealed-secret.yaml
```

### Rotating the SealedSecrets Key

To rotate the encryption key:

```bash
# Delete the old key
kubectl delete secret -n kube-system sealed-secrets-key

# The controller will automatically generate a new key
# Re-encrypt all your sealed secrets with the new key
```

### Backing Up the Key

The SealedSecrets controller stores its private key in a Kubernetes secret. Back it up:

```bash
kubectl get secret -n kube-system sealed-secrets-key -o yaml > sealed-secrets-key-backup.yaml
```

**Important**: Store this backup securely. Without it, you cannot decrypt sealed secrets.

## Verification

Verify that sealed secrets are being decrypted:

```bash
# Check that the regular secret exists
kubectl get secret analytics-secret -n url-shortener

# Compare with the sealed secret
kubectl get sealedsecrets.bitnami.com analytics-secret -n url-shortener
```

## Troubleshooting

### SealedSecret not being decrypted

Check the controller logs:

```bash
kubectl logs -n kube-system deployment/sealed-secrets-controller
```

### kubeseal cannot find the certificate

Ensure the SealedSecrets controller is running:

```bash
kubectl get deployment -n kube-system sealed-secrets-controller
```

Fetch the certificate manually:

```bash
kubeseal --fetch-cert > public-key-cert.pem
```

### Secret not updating

Delete the existing secret and let the controller recreate it:

```bash
kubectl delete secret analytics-secret -n url-shortener
```

## Best Practices

1. **Never commit unencrypted secrets** to Git
2. **Back up the SealedSecrets private key** securely
3. **Rotate keys periodically** for better security
4. **Use different secrets** for different environments (dev, staging, prod)
5. **Limit secret access** using Kubernetes RBAC
6. **Audit secret access** using Kubernetes audit logs

## Migration from Existing Secrets

If you have existing secrets in the cluster:

1. Export the secret:
   ```bash
   kubectl get secret analytics-secret -n url-shortener -o yaml > analytics-secret.yaml
   ```

2. Remove metadata fields (status, creationTimestamp, etc.)

3. Encrypt with kubeseal:
   ```bash
   kubeseal --format yaml < analytics-secret.yaml > analytics-sealed-secret.yaml
   ```

4. Delete the old secret:
   ```bash
   kubectl delete secret analytics-secret -n url-shortener
   ```

5. Apply the sealed secret:
   ```bash
   kubectl apply -f analytics-sealed-secret.yaml
   ```

## Additional Resources

- [SealedSecrets GitHub Repository](https://github.com/bitnami-labs/sealed-secrets)
- [SealedSecrets Documentation](https://sealed-secrets.bitnami.com/)
- [Bitnami SealedSecrets Helm Chart](https://github.com/bitnami/charts/tree/main/bitnami/sealed-secrets)
