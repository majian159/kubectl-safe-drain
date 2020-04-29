package sdrain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (d *Helper) waitMigrate(ctx context.Context, interval, timeout time.Duration, controllers []podController, pods []corev1.Pod, onDoneFn func(controller *podController), globalTimeout time.Duration) error {
	// TODO(justinsb): unnecessary?
	getPodFn := func(namespace, name string) (*corev1.Pod, error) {
		return d.Client.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
	}
	_, err := waitForDelete(ctx, pods, interval, timeout, getPodFn, func(pod *corev1.Pod) {
		fmt.Fprint(d.Out, fmt.Sprintf("pod %s/%s at %s removed\n", pod.Namespace, pod.Name, pod.Spec.NodeName))
	}, globalTimeout)
	if err != nil {
		return err
	}

	err = d.waitForReCreate(ctx, interval, timeout, controllers, onDoneFn, globalTimeout)
	if err != nil {
		return err
	}

	return nil
}

func (d *Helper) waitForReCreate(ctx context.Context, interval, timeout time.Duration, controllers []podController, onDoneFn func(controller *podController), globalTimeout time.Duration) error {
	var controllerTasks = make(map[podController]bool)
	for _, controller := range controllers {
		controllerTasks[controller] = false
	}
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		for controller, complete := range controllerTasks {
			if complete {
				continue
			}
			instance, err := getControllerInstance(d.Client.AppsV1().RESTClient(), controller.Kind+"s", controller.Namespace, controller.Name)
			if err != nil {
				return false, err
			}
			status := instance.Status
			if status.ReadyReplicas == status.Replicas {
				controllerTasks[controller] = true
				if onDoneFn != nil {
					onDoneFn(&controller)
				}
			}
		}

		allComplete := true
		for _, complete := range controllerTasks {
			if !complete {
				allComplete = false
				break
			}
		}
		if !allComplete {
			select {
			case <-ctx.Done():
				return false, fmt.Errorf("global timeout reached: %v", globalTimeout)
			default:
				return false, nil
			}
			return false, nil
		}
		return true, nil
	})
	return err
}

func waitForDelete(ctx context.Context, pods []corev1.Pod, interval, timeout time.Duration, getPodFn func(string, string) (*corev1.Pod, error), onDoneFn func(pod *corev1.Pod), globalTimeout time.Duration) ([]corev1.Pod, error) {
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		pendingPods := []corev1.Pod{}
		for i, pod := range pods {
			p, err := getPodFn(pod.Namespace, pod.Name)
			if apierrors.IsNotFound(err) || (p != nil && p.ObjectMeta.UID != pod.ObjectMeta.UID) {
				if onDoneFn != nil {
					onDoneFn(&pod)
				}
				continue
			} else if err != nil {
				return false, err
			} else {
				pendingPods = append(pendingPods, pods[i])
			}
		}
		pods = pendingPods
		if len(pendingPods) > 0 {
			select {
			case <-ctx.Done():
				return false, fmt.Errorf("global timeout reached: %v", globalTimeout)
			default:
				return false, nil
			}
			return false, nil
		}
		return true, nil
	})
	return pods, err
}

type controllerInstance struct {
	Status controllerStatus
}
type controllerStatus struct {
	Replicas      int
	ReadyReplicas int
}

func getControllerInstance(client rest.Interface, resource, namespace, name string) (*controllerInstance, error) {
	data, err := client.Get().Resource(resource).Namespace(namespace).Name(name).DoRaw()
	if err != nil {
		return nil, err
	}

	var instance controllerInstance
	err = json.Unmarshal(data, &instance)
	if err != nil {
		return nil, err
	}
	return &instance, nil

}
