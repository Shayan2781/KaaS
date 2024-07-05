helm uninstall kaas-api
helm uninstall nginx-ingress
helm uninstall prometheus
helm uninstall grafana
kubectl delete -A ValidatingWebhookConfiguration ingress-nginx-admission
echo "> Cluster Deleted Successfully!"