package agent

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	"deploymate/internal/model"
)

const (
	labelManagedBy   = "app.kubernetes.io/managed-by"
	labelDeployment  = "app.kubernetes.io/part-of"
	labelComponentID = "deploymate.io/deployment-id"
	managedByValue   = "deploymate-agent"
)

// Reconciler converts DeploymentSpec into Kubernetes resources and applies them
// to the cluster via client-go.
type Reconciler struct {
	client kubernetes.Interface
}

// NewReconciler creates a Reconciler backed by the given Kubernetes client.
func NewReconciler(client kubernetes.Interface) *Reconciler {
	return &Reconciler{client: client}
}

// Reconcile ensures the cluster state matches the desired DeploymentSpec.
// It creates or updates a Deployment and a headless Service for the workload.
func (r *Reconciler) Reconcile(ctx context.Context, spec model.DeploymentSpec) error {
	log := log.Ctx(ctx).With().
		Str("deployment_id", spec.ID).
		Str("service", spec.Service).
		Str("image", spec.Image).
		Logger()

	namespace := spec.Environment
	if namespace == "" {
		namespace = "default"
	}

	labels := map[string]string{
		labelManagedBy:   managedByValue,
		labelDeployment:  spec.Service,
		labelComponentID: spec.ID,
	}

	if err := r.ensureNamespace(ctx, namespace); err != nil {
		return fmt.Errorf("ensure namespace %s: %w", namespace, err)
	}

	deploy := buildDeployment(spec, namespace, labels)
	if err := r.applyDeployment(ctx, deploy); err != nil {
		return fmt.Errorf("apply deployment: %w", err)
	}

	svc := buildService(spec, namespace, labels)
	if err := r.applyService(ctx, svc); err != nil {
		return fmt.Errorf("apply service: %w", err)
	}

	log.Info().Str("namespace", namespace).Msg("reconciled successfully")
	return nil
}

// Status reads the current state of the Deployment from the cluster and
// returns a DeploymentStatus summarising its health.
func (r *Reconciler) Status(ctx context.Context, spec model.DeploymentSpec) (model.DeploymentStatus, error) {
	namespace := spec.Environment
	if namespace == "" {
		namespace = "default"
	}

	existing, err := r.client.AppsV1().Deployments(namespace).Get(ctx, spec.ID, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return model.DeploymentStatus{
				ID:    spec.ID,
				Phase: "not_found",
			}, nil
		}
		return model.DeploymentStatus{}, fmt.Errorf("get deployment %s/%s: %w", namespace, spec.ID, err)
	}

	phase := resolvePhase(existing)
	ready := fmt.Sprintf("%d/%d", existing.Status.ReadyReplicas, existing.Status.Replicas)

	return model.DeploymentStatus{
		ID:          spec.ID,
		Phase:       phase,
		Message:     fmt.Sprintf("replicas ready: %s", ready),
		ProgressPct: progressPercent(existing),
	}, nil
}

// Destroy removes the Deployment, Service, and associated resources from the
// cluster. It is idempotent — deleting a non-existent resource is not an error.
func (r *Reconciler) Destroy(ctx context.Context, spec model.DeploymentSpec) error {
	namespace := spec.Environment
	if namespace == "" {
		namespace = "default"
	}

	propagation := metav1.DeletePropagationForeground

	svcErr := r.client.CoreV1().Services(namespace).Delete(ctx, spec.ID, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if svcErr != nil && !errors.IsNotFound(svcErr) {
		return fmt.Errorf("delete service %s/%s: %w", namespace, spec.ID, svcErr)
	}

	deployErr := r.client.AppsV1().Deployments(namespace).Delete(ctx, spec.ID, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if deployErr != nil && !errors.IsNotFound(deployErr) {
		return fmt.Errorf("delete deployment %s/%s: %w", namespace, spec.ID, deployErr)
	}

	log.Ctx(ctx).Info().Str("deployment_id", spec.ID).Str("namespace", namespace).Msg("destroyed")
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (r *Reconciler) ensureNamespace(ctx context.Context, name string) error {
	_, err := r.client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				labelManagedBy: managedByValue,
			},
		},
	}
	_, err = r.client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (r *Reconciler) applyDeployment(ctx context.Context, deploy *appsv1.Deployment) error {
	existing, err := r.client.AppsV1().Deployments(deploy.Namespace).Get(ctx, deploy.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = r.client.AppsV1().Deployments(deploy.Namespace).Create(ctx, deploy, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}

	// Update mutable fields while preserving immutables.
	existing.Labels = deploy.Labels
	existing.Spec.Replicas = deploy.Spec.Replicas
	existing.Spec.Template = deploy.Spec.Template
	_, err = r.client.AppsV1().Deployments(deploy.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func (r *Reconciler) applyService(ctx context.Context, svc *corev1.Service) error {
	_, err := r.client.CoreV1().Services(svc.Namespace).Get(ctx, svc.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = r.client.CoreV1().Services(svc.Namespace).Create(ctx, svc, metav1.CreateOptions{})
		return err
	}
	// Service spec is largely immutable (type, ports, selector).
	// On conflict we just update labels and ignore immutable field changes.
	if err != nil {
		return err
	}
	return nil
}

// buildDeployment translates a DeploymentSpec into a Kubernetes Deployment.
func buildDeployment(spec model.DeploymentSpec, namespace string, labels map[string]string) *appsv1.Deployment {
	replicas := int32(spec.Replicas)
	if replicas == 0 {
		replicas = 1
	}

	container := corev1.Container{
		Name:  spec.Service,
		Image: spec.Image,
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: 8080,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(spec.Resources.CPU),
				corev1.ResourceMemory: resource.MustParse(spec.Resources.Memory),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(spec.Resources.CPU),
				corev1.ResourceMemory: resource.MustParse(spec.Resources.Memory),
			},
		},
		Env: envVarsFromMap(spec.EnvVars),
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.ID,
			Namespace: namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"deploymate.io/org-id":     spec.OrgID,
				"deploymate.io/project-id": spec.ProjectID,
				"deploymate.io/service":    spec.Service,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					labelComponentID: spec.ID,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
				},
			},
		},
	}
}

// buildService translates a DeploymentSpec into a headless ClusterIP Service
// that fronts the deployment.
func buildService(spec model.DeploymentSpec, namespace string, labels map[string]string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.ID,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				labelComponentID: spec.ID,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       8080,
					TargetPort: intstr.FromString("http"),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

func envVarsFromMap(m map[string]string) []corev1.EnvVar {
	if len(m) == 0 {
		return nil
	}
	vars := make([]corev1.EnvVar, 0, len(m))
	for k, v := range m {
		vars = append(vars, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	return vars
}

func resolvePhase(d *appsv1.Deployment) string {
	if d == nil {
		return "unknown"
	}

	for _, cond := range d.Status.Conditions {
		if cond.Type == appsv1.DeploymentProgressing && cond.Status == corev1.ConditionFalse && cond.Reason == "ProgressDeadlineExceeded" {
			return "failed"
		}
		if cond.Type == appsv1.DeploymentAvailable && cond.Status == corev1.ConditionTrue {
			if d.Status.ReadyReplicas == *d.Spec.Replicas {
				return "running"
			}
			return "updating"
		}
	}

	if d.Status.ReadyReplicas == 0 {
		return "pending"
	}
	return "updating"
}

func progressPercent(d *appsv1.Deployment) int {
	if d == nil || d.Spec.Replicas == nil || *d.Spec.Replicas == 0 {
		return 0
	}
	target := *d.Spec.Replicas
	return int((float64(d.Status.ReadyReplicas) / float64(target)) * 100)
}
