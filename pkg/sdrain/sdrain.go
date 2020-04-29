package sdrain

import (
	"context"
	"fmt"
	"io"
	"math"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

type Helper struct {
	Client          kubernetes.Interface
	Force           bool
	Timeout         time.Duration
	DeleteLocalData bool
	Selector        string
	Out             io.Writer
	ErrOut          io.Writer

	DryRun bool
}

// Get pods for nodeName
func (d *Helper) GetPodsForDeletion(nodeName string) (*podDeleteList, []error) {
	podList, err := d.Client.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName}).String()})
	if err != nil {
		return nil, []error{err}
	}

	var pods []podDelete

	for _, pod := range podList.Items {
		var status podDeleteStatus
		for _, filter := range d.makeFilters() {
			status = filter(pod)
			if !status.delete {
				// short-circuit as soon as pod is filtered out
				// at that point, there is no reason to run pod
				// through any additional filters
				break
			}
		}
		if status.delete {
			pods = append(pods, d.newPodDelete(pod, status))
		}
	}

	list := &podDeleteList{items: pods}

	return list, nil
}

func (d *Helper) newPodDelete(pod corev1.Pod, status podDeleteStatus) podDelete {
	controllerRef := metav1.GetControllerOf(&pod)
	var pc *podController
	if controllerRef != nil {
		pc = &podController{
			Kind:      controllerRef.Kind,
			Namespace: pod.Namespace,
			Name:      controllerRef.Name,
		}
		pc, _ = d.getController(*pc)
	}

	return podDelete{
		pod:        pod,
		status:     status,
		controller: pc,
	}
}

func (d *Helper) getController(controller podController) (*podController, error) {
	if controller.Kind != "ReplicaSet" {
		return &controller, nil
	}

	rs, err := d.Client.AppsV1().ReplicaSets(controller.Namespace).Get(controller.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	ref := metav1.GetControllerOf(rs)

	return &podController{
		Kind:      ref.Kind,
		Namespace: controller.Namespace,
		Name:      ref.Name,
	}, nil
}

// Migrate pods to other node
func (d *Helper) MigratePods(list *podDeleteList) error {
	items := list.items
	if len(items) == 0 {
		return nil
	}

	controllerMap := make(map[string]bool)
	var controllers []podController

	for _, item := range items {
		controller := item.controller
		controllerKey := fmt.Sprintf("%s_%s", controller.Kind, controller.Name)
		if controllerMap[controllerKey] {
			continue
		}

		controllers = append(controllers, *item.controller)
		controllerMap[controllerKey] = true
	}

	return d.reCreateControllers(controllers, list.Pods())
}

func (d *Helper) reCreateControllers(controllers []podController, pods []corev1.Pod) error {
	// 0 timeout means infinite, we use MaxInt64 to represent it.
	var globalTimeout time.Duration
	if d.Timeout == 0 {
		globalTimeout = time.Duration(math.MaxInt64)
	} else {
		globalTimeout = d.Timeout
	}

	for _, controller := range controllers {
		err := d.reCreateController(controller)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	ctx := context.TODO()

	return d.waitMigrate(ctx, time.Second, d.Timeout, controllers, pods, func(controller *podController) {
		fmt.Fprint(d.Out, fmt.Sprintf("%s %s/%s pod migrated\n", controller.Kind, controller.Namespace, controller.Name))
	}, globalTimeout)
}
