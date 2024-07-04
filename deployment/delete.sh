helm uninstall kaas-api
helm uninstall nginx-ingress
helm uninstall prometheus
helm uninstall grafana
echo "> Cluster Deleted Successfully!"
#kubectl delete -A ValidatingWebhookConfiguration ingress-nginx-admission

