# apply-helm-infra:
# 	helm repo add bitnami https://charts.bitnami.com/bitnami 2>/dev/null || true
# 	helm repo update
# 	helmfile -f helm/infra/helmfile.yaml sync

apply-helm-sealed-secrets:
	helm install my-sealed-secrets bitnami-labs/sealed-secrets --version 2.18.6 -f ./helm/infra/sealed-secrets/values.yaml -n url-shortener

apply-helm-postgres:
	helm install my-postgres bitnami/postgresql --version 18.6.7 -f ./helm/infra/postgres/values.yaml -n url-shortener

apply-helm-redis:
	helm install my-redis bitnami/redis --version 25.5.3 -f ./helm/infra/redis/values.yaml -n url-shortener

apply-helm-mongodb:
	helm install my-mongodb bitnami/mongodb --version 19.0.7 -f ./helm/infra/mongo/values.yaml -n url-shortener

apply-helm-kafka:
	helm install my-strimzi-kafka-operator oci://quay.io/strimzi-helm/strimzi-kafka-operator --version 1.0.0 -n url-shortener

apply-helm-monitoring:
	helm install monitoring prometheus-community/kube-prometheus-stack -f ./helm/infra/monitoring/values.yaml -n url-shortener

apply-k8s-minikube:
	kubectl apply -k k8s/overlays/minikube

apply-k8s-base:
	kubectl apply -k k8s/base

apply-argo:
	kubectl apply -f root-app