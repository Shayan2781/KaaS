package api

import (
	"KaaS/confgs"
	"KaaS/models"
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

const (
	InternalError   = "Internal server error"
	BadRequest      = "Request body doesn't have correct format"
	SecretExist     = "Secret already exists"
	ConfigExist     = "ConfigMap already exists"
	DeploymentExist = "Deployment already exists"
	ServiceExist    = "Service already exists"
	IngressExist    = "Ingress already exists"
)

type CreateObjectRequest struct {
	AppName        string
	Replicas       int32
	ImageAddress   string
	ImageTag       string
	DomainAddress  string
	ServicePort    int32
	Resources      models.Resource
	Envs           []models.Environment
	ExternalAccess bool
}

type ManagedObjectRequest struct {
	Envs           []models.Environment
	ExternalAccess bool
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
	secret := CreateSecret(secretData, req.AppName)
	_, err = confgs.Client.CoreV1().Secrets("default").Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	configMap := CreateConfigMap(configMapData, req.AppName)
	_, err = confgs.Client.CoreV1().ConfigMaps("default").Create(context.Background(), configMap, metav1.CreateOptions{})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	deployment := CreateDeployment(req.AppName, req.ImageAddress, req.ImageTag, req.Replicas, req.ServicePort, req.Resources, false)
	_, err = confgs.Client.AppsV1().Deployments("default").Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	service := CreateService(req.AppName, req.ServicePort, false)
	_, err = confgs.Client.CoreV1().Services("default").Create(context.Background(), service, metav1.CreateOptions{})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	if req.ExternalAccess {
		ingressObject := CreateIngress(req.AppName, req.DomainAddress, req.ServicePort)
		_, err = confgs.Client.NetworkingV1beta1().Ingresses("default").Create(context.Background(), ingressObject, metav1.CreateOptions{})
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, InternalError)
		}
		host := fmt.Sprintf("for external access domain address is %s.kaas.local", req.DomainAddress)
		return ctx.JSON(http.StatusOK, host)
	} else {
		podName := fmt.Sprintf("for internal access pod name is: %s", req.AppName)
		return ctx.JSON(http.StatusOK, podName)
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
	code := PostgresExistence("", secretData)
	if code == "" {
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	secret := CreateSecret(secretData, code)
	_, err := confgs.Client.CoreV1().Secrets("default").Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	configMap := CreateConfigMap(configMapData, code)
	_, err = confgs.Client.CoreV1().ConfigMaps("default").Create(context.Background(), configMap, metav1.CreateOptions{})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	resource := models.Resource{
		CPU: "500m",
		RAM: "1Gi",
	}
	deployment := CreateDeployment(code, "postgres", "13-alpine", 1, 5432, resource, true)
	_, err = confgs.Client.AppsV1().Deployments("default").Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	service := CreateService(code, 5432, true)
	_, err = confgs.Client.CoreV1().Services("default").Create(context.Background(), service, metav1.CreateOptions{})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	if req.ExternalAccess {
		ingressObject := CreateIngress(code, "", 5432)
		_, err = confgs.Client.NetworkingV1beta1().Ingresses("default").Create(context.Background(), ingressObject, metav1.CreateOptions{})
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, InternalError)
		}
		host := fmt.Sprintf("for external access domain name is postgres.%s.kaas.local", code)
		return ctx.JSON(http.StatusOK, host)
	} else {
		podName := fmt.Sprintf("for internal access pod name is postgres-%s", code)
		return ctx.JSON(http.StatusOK, podName)
	}
}
