# apply-helm-infra:
# 	helm repo add bitnami https://charts.bitnami.com/bitnami 2>/dev/null || true
# 	helm repo update
# 	helmfile -f helm/infra/helmfile.yaml sync

apply-k8s-minikube:
	kubectl apply -k k8s/overlays/minikube

apply-k8s-base:
	kubectl apply -k k8s/base

apply-argo:
	kubectl apply -f root-app