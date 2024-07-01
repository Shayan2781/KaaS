package api

import (
	"KaaS/confgs"
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
	"net/http"
)

func CreateDeployment(req Request) *v1.Deployment {
	deployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-deployment", req.AppName),
		},
		Spec: v1.DeploymentSpec{
			Replicas: &req.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": req.AppName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": req.AppName,
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-config", req.AppName),
									},
								},
							},
						},
						{
							Name: "secret-volume",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: fmt.Sprintf("%s-secret", req.AppName),
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  req.AppName,
							Image: req.ImageAddress,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: req.ServicePort,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{

									corev1.ResourceCPU:    resource.MustParse(req.Resources.CPU),
									corev1.ResourceMemory: resource.MustParse(req.Resources.RAM),
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

func CreateService(req Request) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-service", req.AppName),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": req.AppName,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       req.ServicePort,
					TargetPort: intstr.FromInt32(req.ServicePort),
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

func CreateIngress(req Request) *networkingv1beta1.Ingress {
	ingress := &networkingv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-ingress", req.AppName),
		},
		Spec: networkingv1beta1.IngressSpec{
			Rules: []networkingv1beta1.IngressRule{
				{
					Host: fmt.Sprintf("%s.kaas.local", req.DomainAddress),
					IngressRuleValue: networkingv1beta1.IngressRuleValue{
						HTTP: &networkingv1beta1.HTTPIngressRuleValue{
							Paths: []networkingv1beta1.HTTPIngressPath{
								{
									Path: "/",
									Backend: networkingv1beta1.IngressBackend{
										ServiceName: fmt.Sprintf("%s-service", req.AppName),
										ServicePort: intstr.FromInt32(req.ServicePort),
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

func CheckExistence(ctx echo.Context, req Request) error {
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
