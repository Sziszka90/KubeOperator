package controllers

import (
	"fmt"
	"net"

	webappv1 "gomodule/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *AppRunnerReconciler) desiredDeployment(apprunner webappv1.AppRunner) (appsv1.Deployment, error) {
	depl := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      apprunner.Name,
			Namespace: apprunner.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: apprunner.Spec.Frontend.Replicas, // won't be nil because defaulting
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"operator": apprunner.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"operator": apprunner.Name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "container",
							Image: apprunner.Spec.Frontend.Image,
							Ports: []corev1.ContainerPort{
								{ContainerPort: 80, Name: "http", Protocol: "TCP"},
							},
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(&apprunner, &depl, r.Scheme); err != nil {
		return depl, err
	}

	return depl, nil
}

func (r *AppRunnerReconciler) desiredService(apprunner webappv1.AppRunner) (corev1.Service, error) {
	svc := corev1.Service{
		TypeMeta: metav1.TypeMeta{APIVersion: corev1.SchemeGroupVersion.String(), Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      apprunner.Name,
			Namespace: apprunner.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Name: "http", Port: 8080, Protocol: "TCP", TargetPort: intstr.FromString("http")},
			},
			Selector: map[string]string{"operator": apprunner.Name},
			Type:     corev1.ServiceTypeLoadBalancer,
		},
	}

	// always set the controller reference so that we know which object owns this.
	if err := ctrl.SetControllerReference(&apprunner, &svc, r.Scheme); err != nil {
		return svc, err
	}

	return svc, nil
}

func urlForService(svc corev1.Service, port int32) string {
	// notice that we unset this if it's not present -- we always want the
	// state to reflect what we observe.
	if len(svc.Status.LoadBalancer.Ingress) == 0 {
		return ""
	}

	host := svc.Status.LoadBalancer.Ingress[0].Hostname
	if host == "" {
		host = svc.Status.LoadBalancer.Ingress[0].IP
	}

	return fmt.Sprintf("http://%s", net.JoinHostPort(host, fmt.Sprintf("%v", port)))
}
