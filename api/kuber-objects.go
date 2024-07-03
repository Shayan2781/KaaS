package api

import (
	"KaaS/configs"
	"KaaS/models"
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	"io"
	v1 "k8s.io/api/apps/v1"
	cronv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

const (
	letterBytes  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	specialBytes = "!@#$%^&*()_+-=[]{}\\|;':\",.<>/?`~"
	numBytes     = "0123456789"
)

func CreateDeployment(configMapData, secretData map[string]string, name, imageAddress, imageTag string, replicas, servicePort int32, containerResource models.Resource, managed, monitor bool) *v1.Deployment {
	if managed {
		name = fmt.Sprintf("postgres-%s", name)
	}
	monitorLabel := "false"
	if monitor {
		monitorLabel = "true"
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
						"app":     name,
						"monitor": monitorLabel,
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
							//EnvFrom: []corev1.EnvFromSource{
							//	{
							//		ConfigMapRef: &corev1.ConfigMapEnvSource{
							//			LocalObjectReference: corev1.LocalObjectReference{
							//				Name: fmt.Sprintf("%s-config", name),
							//			},
							//		},
							//	},
							//	{
							//		SecretRef: &corev1.SecretEnvSource{
							//			LocalObjectReference: corev1.LocalObjectReference{
							//				Name: fmt.Sprintf("%s-secret", name),
							//			},
							//		},
							//	},
							//},
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
	envs := make([]corev1.EnvVar, 0)
	for configKey := range configMapData {
		envs = append(envs, corev1.EnvVar{
			Name:  configKey,
			Value: configMapData[configKey],
		})
	}
	for secretKey := range secretData {
		envs = append(envs, corev1.EnvVar{
			Name:  secretKey,
			Value: configMapData[secretKey],
		})
	}
	deployment.Spec.Template.Spec.Containers[0].Env = envs
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
					Protocol:   corev1.ProtocolTCP,
					Port:       int32(80),
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

func CreateSecret(secretData map[string]string, secretName string, managed bool) *corev1.Secret {
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
		StringData: secretData,
	}
	return secret
}

func CreateCronJob(appName, imageAddress, imageTag string, servicePort int32) *cronv1.CronJob {
	//servicePortStr := strconv.Itoa(int(servicePort))
	serviceName := fmt.Sprintf("%s-service", appName)
	cronJob := &cronv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-cronjob", appName),
		},
		Spec: cronv1.CronJobSpec{
			Schedule: "* * * * *",
			JobTemplate: cronv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"created_by": fmt.Sprintf("%s-cronjob", appName),
					},
				},
				Spec: cronv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:    appName,
									Image:   fmt.Sprintf("%s:%s", imageAddress, imageTag),
									Command: []string{"sh", "-c", fmt.Sprintf("while true; do curl http://%s:80/healthz | grep \"^HTTP/\"; sleep 5; done", serviceName)},
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	}
	return cronJob
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

func CheckExistence(ctx echo.Context, appName string) error {
	_, err := configs.Client.CoreV1().Secrets("default").Get(context.Background(), fmt.Sprintf("%s-secret", appName), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
	} else {
		return ctx.JSON(http.StatusNotAcceptable, ObjectExist)
	}
	return nil
}

func PostgresExistence(configData map[string]string) string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	random := 10000 + rand.Intn(89999)
	newCode := strconv.Itoa(random)
	_, err := configs.Client.CoreV1().ConfigMaps("default").Get(context.Background(), fmt.Sprintf("postgres-%s-config", newCode), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return newCode
		}
	} else {
		return PostgresExistence(configData)
	}
	return ""
}

func GeneratePassword(length int, useLetters bool, useSpecial bool, useNum bool) string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		if useLetters {
			b[i] = letterBytes[rand.Intn(len(letterBytes))]
		} else if useSpecial {
			b[i] = specialBytes[rand.Intn(len(specialBytes))]
		} else if useNum {
			b[i] = numBytes[rand.Intn(len(numBytes))]
		}
	}
	return string(b)
}

func GetJobsLogs(appName string) {
	counter := 0
	//_, err := configs.Client.BatchV1().CronJobs("default").Get(context.Background(), fmt.Sprintf("%s-cronjob", appName), metav1.GetOptions{})
	//if err != nil {
	//	log.Printf("failed to get %s-cronjob: %v", appName, err)
	//}
	for {
		jobs, jobErr := configs.Client.BatchV1().Jobs("default").List(context.Background(), metav1.ListOptions{})
		if jobErr != nil {
			log.Printf("failed to get jobs for %s-cronjob: %v", appName, jobErr)
		}
		var filteredJobs []*cronv1.Job
		for _, job := range jobs.Items {
			if job.Labels["created_by"] == fmt.Sprintf("%s-cronjob", appName) {
				filteredJobs = append(filteredJobs, &job)
			}
		}
		fmt.Println("length job", len(filteredJobs))
		for _, job := range filteredJobs {
			pods, podErr := configs.Client.CoreV1().Pods(job.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
			})
			if podErr != nil {
				log.Printf("failed to get pods for job %s: %v", job.Name, podErr)
			}
			fmt.Println("length pod", len(pods.Items))
			for i, pod := range pods.Items {
				stdout, logErr := configs.Client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).Stream(context.Background())
				if logErr != nil {
					log.Printf("failed to get logs for pod %s: %v", pod.Name, logErr)
				}
				content, logErr := io.ReadAll(stdout)
				if logErr != nil {
					log.Printf("failed to read content: %v", logErr)
				}
				println(counter)
				counter++
				fmt.Println(i, string(content))
			}
			//err = configs.Client.BatchV1().Jobs(job.Namespace).Delete(context.Background(), job.Name, metav1.DeleteOptions{
			//	GracePeriodSeconds: ptr.To(int64(0)),
			//	PropagationPolicy:  ptr.To(metav1.DeletePropagationBackground),
			//})
			//if err != nil {
			//	log.Printf("failed to delete job %s: %v", job.Name, podErr)
			//}
		}
		time.Sleep(time.Second * 5)
	}
}
