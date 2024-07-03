minikube addons enable registry
docker run --rm -it --network=host alpine ash -c "apk add socat && socat TCP-LISTEN:5000,reuseaddr,fork TCP:$(minikube ip):5000"
kubectl port-forward --namespace kube-system service/registry 5000:80
docker push localhost:5000/kaas-api-webserver