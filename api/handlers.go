package api

import (
	"KaaS/configs"
	"KaaS/models"
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
)

const (
	InternalError       = "Internal server error"
	BadRequest          = "Request body doesn't have correct format"
	ObjectExist         = "Object already exists"
	DeploymentExistence = "Deployment doesn't exist"
)

type CreateObjectRequest struct {
	AppName        string               `json:"AppName"`
	Replicas       int32                `json:"Replicas"`
	ImageAddress   string               `json:"ImageAddress"`
	ImageTag       string               `json:"ImageTag"`
	DomainAddress  string               `json:"DomainAddress"`
	ServicePort    int32                `json:"ServicePort"`
	Resources      models.Resource      `json:"Resources"`
	Envs           []models.Environment `json:"Envs"`
	ExternalAccess bool                 `json:"ExternalAccess"`
}

type ManagedObjectRequest struct {
	Envs           []models.Environment
	ExternalAccess bool
}

type GetDeploymentResponse struct {
	DeploymentName string             `json:"DeploymentName"`
	Replicas       int32              `json:"Replicas"`
	ReadyReplicas  int32              `json:"ReadyReplicas"`
	PodStatuses    []models.PodStatus `json:"PodStatuses"`
}

func DeployUnmanagedObjects(ctx echo.Context) error {
	req := new(CreateObjectRequest)
	if err := ctx.Bind(req); err != nil {
		return ctx.JSON(http.StatusBadRequest, BadRequest)
	}
	err := CheckExistence(ctx, *req)
	if err != nil {
		return err
	}
	secretData := make(map[string][]byte)
	configMapData := make(map[string]string)
	for _, env := range req.Envs {
		if env.IsSecret {
			secretData[env.Key] = []byte(env.Value)
		} else {
			configMapData[env.Key] = env.Value
		}
	}
	appName := strings.ToLower(req.AppName)
	secret := CreateSecret(secretData, appName, false)
	_, err = configs.Client.CoreV1().Secrets("default").Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		println(err.Error())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	configMap := CreateConfigMap(configMapData, appName, false)
	_, err = configs.Client.CoreV1().ConfigMaps("default").Create(context.Background(), configMap, metav1.CreateOptions{})
	if err != nil {
		println(err.Error())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	deployment := CreateDeployment(appName, req.ImageAddress, req.ImageTag, req.Replicas, req.ServicePort, req.Resources, false)
	_, err = configs.Client.AppsV1().Deployments("default").Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		println(err.Error())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	service := CreateService(appName, req.ServicePort, false)
	_, err = configs.Client.CoreV1().Services("default").Create(context.Background(), service, metav1.CreateOptions{})
	if err != nil {
		println("service: ", err.Error())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	if req.ExternalAccess {
		ingressObject := CreateIngress(appName, req.ServicePort, false)
		_, err = configs.Client.NetworkingV1().Ingresses("default").Create(context.Background(), ingressObject, metav1.CreateOptions{})
		if err != nil {
			println("ingress: ", err.Error())
			return ctx.JSON(http.StatusInternalServerError, InternalError)
		}
		host := fmt.Sprintf("for external access, domain address is: %s.kaas.local", req.AppName)
		return ctx.JSON(http.StatusOK, host)
	} else {
		serviceName := fmt.Sprintf("for internal access, service name is: %s-service", req.AppName)
		return ctx.JSON(http.StatusOK, serviceName)
	}
}

func DeployManagedObjects(ctx echo.Context) error {
	req := new(ManagedObjectRequest)
	if err := ctx.Bind(req); err != nil {
		return ctx.JSON(http.StatusBadRequest, BadRequest)
	}
	secretData := make(map[string][]byte)
	configMapData := make(map[string]string)
	for _, env := range req.Envs {
		if env.IsSecret {
			secretData[env.Key] = []byte(env.Value)
		} else {
			configMapData[env.Key] = env.Value
		}
	}
	code := PostgresExistence(secretData)
	if code == "" {
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	secret := CreateSecret(secretData, code, true)
	_, err := configs.Client.CoreV1().Secrets("default").Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		println("secret:", err)
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	configMap := CreateConfigMap(configMapData, code, true)
	_, err = configs.Client.CoreV1().ConfigMaps("default").Create(context.Background(), configMap, metav1.CreateOptions{})
	if err != nil {
		println("config:", err)
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	resource := models.Resource{
		CPU: "500m",
		RAM: "1Gi",
	}
	deployment := CreateDeployment(code, "postgres", "13-alpine", 1, 5432, resource, true)
	_, err = configs.Client.AppsV1().Deployments("default").Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		println("deployment:", err)
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	service := CreateService(code, 5432, true)
	_, err = configs.Client.CoreV1().Services("default").Create(context.Background(), service, metav1.CreateOptions{})
	if err != nil {
		println("service:", err.Error())
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	if req.ExternalAccess {
		ingressObject := CreateIngress(code, 5432, true)
		_, err = configs.Client.NetworkingV1().Ingresses("default").Create(context.Background(), ingressObject, metav1.CreateOptions{})
		if err != nil {
			println("ingress:", err.Error())
			return ctx.JSON(http.StatusInternalServerError, InternalError)
		}
		host := fmt.Sprintf("for external access, domain name is: postgres.%s.kaas.local", code)
		return ctx.JSON(http.StatusOK, host)
	} else {
		serviceName := fmt.Sprintf("for internal access, service name: is postgres-%s-service", code)
		return ctx.JSON(http.StatusOK, serviceName)
	}
}

func GetDeployment(ctx echo.Context) error {
	res := GetDeploymentResponse{}
	appName := ctx.Param("app-name")
	deployment, err := configs.Client.AppsV1().Deployments("default").Get(context.Background(), fmt.Sprintf("%s-deployment", appName), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return ctx.JSON(http.StatusNotAcceptable, DeploymentExistence)
		}
	}
	pods, err := configs.Client.CoreV1().Pods("default").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	var filteredPods []*v1.Pod
	for _, pod := range pods.Items {
		if pod.Labels["app"] == appName {
			filteredPods = append(filteredPods, &pod)
		}
	}
	podStatuses := make([]models.PodStatus, len(filteredPods))
	for i, pod := range filteredPods {
		podStatuses[i] = models.PodStatus{
			Name:      pod.Name,
			Phase:     string(pod.Status.Phase),
			HostID:    pod.Status.HostIP,
			PodIP:     pod.Status.PodIP,
			StartTime: pod.Status.StartTime.String(),
		}
	}
	res.DeploymentName = deployment.Name
	res.Replicas = deployment.Status.Replicas
	res.ReadyReplicas = deployment.Status.ReadyReplicas
	res.PodStatuses = podStatuses
	return ctx.JSON(http.StatusOK, res)
}

func GetAllDeployments(ctx echo.Context) error {
	var res []GetDeploymentResponse
	deployments, err := configs.Client.AppsV1().Deployments("default").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	pods, err := configs.Client.CoreV1().Pods("default").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	res = make([]GetDeploymentResponse, len(deployments.Items))
	for i, deployment := range deployments.Items {
		var filteredPods []*v1.Pod
		for _, pod := range pods.Items {
			deploymentName := fmt.Sprintf("%s-deployment", pod.Labels["app"])
			if deploymentName == deployment.Name {
				filteredPods = append(filteredPods, &pod)
			}
		}
		podStatuses := make([]models.PodStatus, len(filteredPods))
		for j, pod := range filteredPods {
			podStatuses[j] = models.PodStatus{
				Name:      pod.Name,
				Phase:     string(pod.Status.Phase),
				HostID:    pod.Status.HostIP,
				PodIP:     pod.Status.PodIP,
				StartTime: pod.Status.StartTime.String(),
			}
		}
		res[i].DeploymentName = deployment.Name
		res[i].Replicas = deployment.Status.Replicas
		res[i].ReadyReplicas = deployment.Status.ReadyReplicas
		res[i].PodStatuses = podStatuses
	}
	return ctx.JSON(http.StatusOK, res)
}
