# KaaS - Kubernetes as a Service

A RESTful API service built with Go that simplifies Kubernetes deployments by providing an abstraction layer for managing containerized applications. Deploy your containers to Kubernetes without writing YAML manifests.

## Features

- **Unmanaged Deployments**: Deploy any Docker container with custom configurations
- **Managed Deployments**: One-click PostgreSQL database provisioning with auto-generated credentials
- **Automatic Resource Creation**: Automatically creates Deployments, Services, ConfigMaps, Secrets, and Ingress resources
- **External Access**: Optional ingress configuration for external access via `*.kaas.local` domain
- **Resource Limits**: Configure CPU and memory limits for containers
- **Environment Variables**: Support for both regular environment variables and secrets
- **Deployment Monitoring**: Query deployment status and pod information

## Tech Stack

- **Language**: Go 1.22
- **Web Framework**: [Echo v4](https://echo.labstack.com/)
- **Kubernetes Client**: [client-go](https://github.com/kubernetes/client-go)
- **Container Orchestration**: Kubernetes / Minikube
- **Monitoring**: Prometheus + Grafana
- **Deployment**: Helm Charts

## Prerequisites

- Go 1.22+
- Kubernetes cluster (Minikube recommended for local development)
- kubectl configured with cluster access
- Helm 3.x (for deployment)
- Docker (for building images)

## Project Structure

```
KaaS/
├── main.go                 # Application entry point
├── api/
│   ├── handlers.go         # HTTP request handlers
│   ├── routes.go           # API route definitions
│   └── kuber-objects.go    # Kubernetes object builders
├── configs/
│   └── kubernetes.go       # Kubernetes client configuration
├── models/
│   └── models.go           # Data models
├── deployment/
│   ├── setup.sh            # Cluster setup script
│   ├── delete.sh           # Cluster teardown script
│   └── kaas-api/           # Helm chart
│       ├── Chart.yaml
│       ├── values.yaml
│       ├── prometheus.yaml
│       └── templates/
└── dockerfile              # Container build file
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/deploy-unmanaged` | Deploy a custom container application |
| `POST` | `/deploy-managed` | Deploy a managed PostgreSQL instance |
| `GET` | `/get-deployment/:app-name` | Get deployment status by app name |
| `GET` | `/get-all-deployments` | List all deployments |

## API Reference

### Deploy Unmanaged Application

Deploy any Docker container with custom configuration.

**Endpoint**: `POST /deploy-unmanaged`

**Request Body**:
```json
{
  "AppName": "my-app",
  "Replicas": 3,
  "ImageAddress": "nginx",
  "ImageTag": "latest",
  "DomainAddress": "my-app.example.com",
  "ServicePort": 80,
  "Resources": {
    "CPU": "500m",
    "RAM": "256Mi"
  },
  "Envs": [
    {
      "Key": "DATABASE_URL",
      "Value": "postgres://...",
      "IsSecret": true
    },
    {
      "Key": "LOG_LEVEL",
      "Value": "info",
      "IsSecret": false
    }
  ],
  "ExternalAccess": true
}
```

**Response** (with ExternalAccess=true):
```json
"for external access, domain address is: my-app.kaas.local"
```

**Response** (with ExternalAccess=false):
```json
"for internal access, service name is: my-app-service"
```

**Created Resources**:
- Deployment: `{app-name}-deployment`
- Service: `{app-name}-service`
- ConfigMap: `{app-name}-config`
- Secret: `{app-name}-secret`
- Ingress (optional): `{app-name}-ingress`

---

### Deploy Managed PostgreSQL

Deploy a pre-configured PostgreSQL database instance.

**Endpoint**: `POST /deploy-managed`

**Request Body**:
```json
{
  "Envs": [
    {
      "Key": "POSTGRES_USER",
      "Value": "admin",
      "IsSecret": true
    },
    {
      "Key": "POSTGRES_PASSWORD",
      "Value": "secretpassword",
      "IsSecret": true
    },
    {
      "Key": "POSTGRES_DB",
      "Value": "mydb",
      "IsSecret": false
    }
  ],
  "ExternalAccess": false
}
```

**Response**:
```json
"for internal access, service name: is postgres-{code}-service"
```

*Note: A unique code is auto-generated for each managed instance.*

**Default Resources**:
- Image: `postgres:13-alpine`
- CPU: `500m`
- Memory: `1Gi`
- Port: `5432`

---

### Get Deployment Status

**Endpoint**: `GET /get-deployment/:app-name`

**Response**:
```json
{
  "DeploymentName": "my-app-deployment",
  "Replicas": 3,
  "ReadyReplicas": 3,
  "PodStatuses": [
    {
      "Name": "my-app-deployment-abc123",
      "Phase": "Running",
      "HostID": "192.168.49.2",
      "PodIP": "10.244.0.15",
      "StartTime": "2024-01-15 10:30:00 +0000 UTC"
    }
  ]
}
```

---

### Get All Deployments

**Endpoint**: `GET /get-all-deployments`

**Response**:
```json
[
  {
    "DeploymentName": "my-app-deployment",
    "Replicas": 3,
    "ReadyReplicas": 3,
    "PodStatuses": [...]
  },
  {
    "DeploymentName": "postgres-12345-deployment",
    "Replicas": 1,
    "ReadyReplicas": 1,
    "PodStatuses": [...]
  }
]
```

## Getting Started

### Local Development

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd KaaS
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Start Minikube** (if not already running):
   ```bash
   minikube start
   ```

4. **Run the application**:
   ```bash
   go run main.go
   ```

   The API server will start at `http://localhost:8080`

### Deployment to Kubernetes

1. **Start Minikube tunnel** (for ingress access):
   ```bash
   minikube tunnel
   ```

2. **Deploy using Helm**:
   ```bash
   cd deployment
   ./setup.sh
   ```

   This will install:
   - KaaS API
   - NGINX Ingress Controller
   - Prometheus (monitoring)
   - Grafana (visualization)

3. **Teardown the cluster**:
   ```bash
   ./delete.sh
   ```

### Building Docker Image

```bash
docker build -t kaas-api:latest .
```

## Monitoring Setup

The project includes Prometheus and Grafana for monitoring.

### Install Prometheus

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install -f deployment/kaas-api/prometheus.yaml prometheus prometheus-community/prometheus
kubectl expose service prometheus-server --type=NodePort --target-port=9090 --name=prometheus-server-ext
minikube service prometheus-server-ext
```

### Install Grafana

```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
helm install grafana grafana/grafana
kubectl expose service grafana --type=NodePort --target-port=3000 --name=grafana-ext
minikube service grafana-ext
```

**Get Grafana admin password**:
```bash
kubectl get secret --namespace default grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo
```

## Configuration

### Helm Values

Key configuration options in `deployment/kaas-api/values.yaml`:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Docker image repository | `localhost:5000/nodejs-test:0.0.1` |
| `image.port` | Container port | `3000` |
| `replicaCount` | Number of replicas | `1` |
| `autoScaling.maxReplicas` | Max replicas for HPA | `10` |
| `autoScaling.averageCPU` | Target CPU for autoscaling | `800m` |
| `service.type` | Kubernetes service type | `ClusterIP` |
| `ingress.host` | Ingress hostname | `kaas-api.example` |

## Error Responses

| Status Code | Message | Description |
|-------------|---------|-------------|
| `400` | Request body doesn't have correct format | Invalid JSON or missing fields |
| `406` | Object already exists | Resource with same name already deployed |
| `406` | Deployment doesn't exist | Requested deployment not found |
| `500` | Internal server error | Kubernetes API error |

## License

This project was created as part of a Cloud Computing course final project.
