package api

import (
	"KaaS/confgs"
	"KaaS/models"
	"context"
	"github.com/labstack/echo/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

const (
	InternalError       = "Internal server error"
	BadRequest          = "Request body doesn't have correct format"
	SecretExist         = "Secret already exists"
	ConfigExist         = "ConfigMap already exists"
	DeploymentExist     = "Deployment already exists"
	ServiceExist        = "Service already exists"
	IngressExist        = "Ingress already exists"
	CreatedSuccessfully = "All objects created successfully"
)

type Request struct {
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

func CreateK8SObjects(ctx echo.Context) error {
	req := new(Request)
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
	deployment := CreateDeployment(*req)
	_, err = confgs.Client.AppsV1().Deployments("default").Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	service := CreateService(*req)
	_, err = confgs.Client.CoreV1().Services("default").Create(context.Background(), service, metav1.CreateOptions{})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	ingressObject := CreateIngress(*req)
	_, err = confgs.Client.NetworkingV1beta1().Ingresses("default").Create(context.Background(), ingressObject, metav1.CreateOptions{})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, InternalError)
	}
	return ctx.JSON(http.StatusOK, CreatedSuccessfully)
}
