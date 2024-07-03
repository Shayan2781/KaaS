1. helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
2. helm repo update
3. helm install -f values.yaml prometheus prometheus-community/prometheus
4. kubectl expose service prometheus-server --type=NodePort --target-port=9090 --name=prometheus-server-ext
5. minikube service prometheus-server-ext
6. helm repo add grafana https://grafana.github.io/helm-charts 
7. helm repo update
8. helm install grafana grafana/grafana
9. kubectl expose service grafana --type=NodePort --target-port=3000 --name=grafana-ext
10. minikube service grafana-ext
11. kubectl get secret --namespace default grafana -o jsonpath="{.data.admin-password}" | ForEach-Object { [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($_)) }
