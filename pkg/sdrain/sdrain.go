package sdrain

import (
	"io"

	"k8s.io/client-go/kubernetes"
)

type Helper struct {
	Client   kubernetes.Interface
	Selector string
	Out      io.Writer
	ErrOut   io.Writer

	DryRun bool
}
