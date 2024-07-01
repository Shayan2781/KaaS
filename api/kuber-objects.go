package api

import (
	"KaaS/confgs"
	"KaaS/models"
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
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
	appName := ""
	if managed {
		appName = fmt.Sprintf("postgres-%s", name)
	}
	deployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-deployment", name),
		},
		Spec: v1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": appName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": appName,
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-config", name),
									},
								},
							},
						},
						{
							Name: "secret-volume",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: fmt.Sprintf("%s-secret", name),
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  appName,
							Image: fmt.Sprintf("%s:%s", imageAddress, imageTag),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: servicePort,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{

									corev1.ResourceCPU:    resource.MustParse(containerResource.CPU),
									corev1.ResourceMemory: resource.MustParse(containerResource.RAM),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config-volume",
									MountPath: "/etc/config",
								},
								{
									Name:      "secret-volume",
									MountPath: "/etc/secret",
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
	appName := ""
	if managed {
		appName = fmt.Sprintf("postgres-%s", name)
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-service", name),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": appName,
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

func CreateConfigMap(configMapData map[string]string, configName string) *corev1.ConfigMap {
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

func CreateSecret(secretData map[string][]byte, secretName string) *corev1.Secret {
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

func CreateIngress(name, domainAddress string, servicePort int32) *networkingv1beta1.Ingress {
	host := ""
	if domainAddress == "" {
		host = fmt.Sprintf("postgres.%s", name)
	} else {
		host = domainAddress
	}
	ingress := &networkingv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-ingress", name),
		},
		Spec: networkingv1beta1.IngressSpec{
			Rules: []networkingv1beta1.IngressRule{
				{
					Host: fmt.Sprintf("%s.kaas.local", host),
					IngressRuleValue: networkingv1beta1.IngressRuleValue{
						HTTP: &networkingv1beta1.HTTPIngressRuleValue{
							Paths: []networkingv1beta1.HTTPIngressPath{
								{
									Path: "/",
									Backend: networkingv1beta1.IngressBackend{
										ServiceName: fmt.Sprintf("%s-service", name),
										ServicePort: intstr.FromInt32(servicePort),
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
	_, err := confgs.Client.CoreV1().Secrets("default").Get(context.Background(), fmt.Sprintf("%s-secret", req.AppName), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return ctx.JSON(http.StatusNotAcceptable, SecretExist)
		} else {
			return ctx.JSON(http.StatusInternalServerError, InternalError)
		}
	}
	_, err = confgs.Client.CoreV1().ConfigMaps("default").Get(context.Background(), fmt.Sprintf("%s-config", req.AppName), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return ctx.JSON(http.StatusNotAcceptable, ConfigExist)
		} else {
			return ctx.JSON(http.StatusInternalServerError, InternalError)
		}
	}
	_, err = confgs.Client.AppsV1().Deployments("default").Get(context.Background(), fmt.Sprintf("%s-deployment", req.AppName), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return ctx.JSON(http.StatusNotAcceptable, DeploymentExist)
		} else {
			return ctx.JSON(http.StatusInternalServerError, InternalError)
		}
	}
	_, err = confgs.Client.CoreV1().Services("default").Get(context.Background(), fmt.Sprintf("%s-service", req.AppName), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return ctx.JSON(http.StatusNotAcceptable, ServiceExist)
		} else {
			return ctx.JSON(http.StatusInternalServerError, InternalError)
		}
	}
	_, err = confgs.Client.NetworkingV1().Ingresses("default").Get(context.Background(), fmt.Sprintf("%s-ingress", req.AppName), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return ctx.JSON(http.StatusNotAcceptable, IngressExist)
		} else {
			return ctx.JSON(http.StatusInternalServerError, InternalError)
		}
	}
	return nil
}

func PostgresExistence(code string, secretData map[string][]byte) string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	random := 10000 + rand.Intn(89999)
	newCode := strconv.Itoa(random)
	_, err := confgs.Client.CoreV1().Secrets("default").Get(context.Background(), fmt.Sprintf("%s-secret", code), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return PostgresExistence(newCode, secretData)
		} else {
			return ""
		}
	}
	return newCode
}
