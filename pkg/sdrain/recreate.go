package sdrain

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"
)

func (d *Helper) reCreateController(controller podController) error {
	switch controller.Kind {
	case "Deployment":
		return d.reCreateDeployment(controller.Namespace, controller.Name)
	case "StatefulSet":
		return d.reCreateStatefulSet(controller.Namespace, controller.Name)
	}

	// todo: not support
	return nil
}

func (d *Helper) reCreate(resource, namespace, controllerName string, body []byte) error {
	restClient := d.Client.AppsV1().RESTClient()
	return restClient.Patch(types.MergePatchType).
		Resource(resource).
		Namespace(namespace).
		Name(controllerName).
		Body(body).
		Do().
		Error()
}

func (d *Helper) reCreateDeployment(namespace, controllerName string) error {
	return d.reCreate("deployments", namespace, controllerName, []byte(fmt.Sprintf(`{
	"spec": {
		"template": {
			"metadata": {
				"annotations": {
					"fleet.io/safe-drain": "%d"
				}
			}
		}
	}
}`, time.Now().Unix())))
}

func (d *Helper) reCreateStatefulSet(namespace, controllerName string) error {
	return d.reCreate("statefulsets", namespace, controllerName, []byte(fmt.Sprintf(`{
					"spec": {
						"template": {
							"metadata": {
								"annotations": {
									"fleet.io/safe-drain": "%d"
								}
							}
						}
					}
				}`, time.Now().Unix())))
}
