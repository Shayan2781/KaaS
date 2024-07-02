package api

import (
	"KaaS/configs"
	"KaaS/models"
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

func CreateDeployment(name, imageAddress, imageTag string, replicas, servicePort int32, containerResource models.Resource, managed bool) *v1.Deployment {
	if managed {
		name = fmt.Sprintf("postgres-%s", name)
	}
	deployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-deployment", name),
		},
		Spec: v1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: fmt.Sprintf("%s:%s", imageAddress, imageTag),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: servicePort,
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: fmt.Sprintf("%s-config", name),
										},
									},
								},
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: fmt.Sprintf("%s-secret", name),
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(containerResource.CPU),
									corev1.ResourceMemory: resource.MustParse(containerResource.RAM),
								},
							},
						},
					},
				},
			},
		},
	}
	return deployment
}

func CreateService(name string, servicePort int32, managed bool) *corev1.Service {
	if managed {
		name = fmt.Sprintf("postgres-%s", name)
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-service", name),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": name,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       servicePort,
					TargetPort: intstr.FromInt32(servicePort),
				},
			},
		},
	}
	return service
}

func CreateConfigMap(configMapData map[string]string, configName string, managed bool) *corev1.ConfigMap {
	if managed {
		configName = fmt.Sprintf("postgres-%s", configName)
	}
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-config", configName),
		},
		Data: configMapData,
	}
	return configMap
}

func CreateSecret(secretData map[string][]byte, secretName string, managed bool) *corev1.Secret {
	if managed {
		secretName = fmt.Sprintf("postgres-%s", secretName)
	}
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-secret", secretName),
		},
		Data: secretData,
	}
	return secret
}

func CreateIngress(name string, servicePort int32, managed bool) *networkingv1.Ingress {
	host := ""
	if managed {
		host = fmt.Sprintf("postgres.%s", name)
		name = fmt.Sprintf("postgres-%s", name)
	} else {
		host = name
	}
	pathType := networkingv1.PathTypePrefix
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-ingress", name),
			Namespace: "default",
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: fmt.Sprintf("%s.kaas.local", host),
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									PathType: &pathType,
									Path:     "/",
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: fmt.Sprintf("%s-service", name),
											Port: networkingv1.ServiceBackendPort{
												Number: servicePort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return ingress
}

func CheckExistence(ctx echo.Context, req CreateObjectRequest) error {
	_, err := configs.Client.CoreV1().Secrets("default").Get(context.Background(), fmt.Sprintf("%s-secret", req.AppName), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
	} else {
		return ctx.JSON(http.StatusNotAcceptable, ObjectExist)
	}
	return nil
}

func PostgresExistence(secretData map[string][]byte) string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	random := 10000 + rand.Intn(89999)
	newCode := strconv.Itoa(random)
	_, err := configs.Client.CoreV1().Secrets("default").Get(context.Background(), fmt.Sprintf("postgres-%s-secret", newCode), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return newCode
		}
	} else {
		return PostgresExistence(secretData)
	}
	return ""
}
