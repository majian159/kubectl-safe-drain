package sdrain

import (
	"kubectl-sdrain/pkg/sdrain"

	"k8s.io/kubectl/pkg/util/templates"

	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/spf13/cobra"

	"k8s.io/kubectl/pkg/scheme"

	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
)

type SafeDrainCmdOptions struct {
	PrintFlags *genericclioptions.PrintFlags
	ToPrinter  func(string) (printers.ResourcePrinterFunc, error)

	safeDrainer *sdrain.Helper

	genericclioptions.IOStreams
}

var (
	safeDrainLong = templates.LongDesc(i18n.T(`
		Safe drain node in preparation for maintenance.
		The given node will be marked unschedulable to prevent new pods from arriving.`))

	safeDrainExample = templates.Examples(i18n.T(`
		# Safe drain node "foo", even if there are pods not managed by a ReplicationController, ReplicaSet, Job or StatefulSet on it.
		$ kubectl sdrain foo`))
)

func NewDrainCmdOptions(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) *SafeDrainCmdOptions {
	o := &SafeDrainCmdOptions{
		PrintFlags: genericclioptions.NewPrintFlags("sdrained").WithTypeSetter(scheme.Scheme),
		IOStreams:  ioStreams,
		safeDrainer: &sdrain.Helper{
			Out:    ioStreams.Out,
			ErrOut: ioStreams.ErrOut,
		},
	}
	return o
}

func NewCmdSafeDrain(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) *cobra.Command {
	o := NewDrainCmdOptions(f, ioStreams)

	cmd := &cobra.Command{
		Use:                   "sdrain NODE",
		DisableFlagsInUseLine: true,
		Short:                 i18n.T("Safe drain node in preparation for maintenance"),
		Long:                  safeDrainLong,
		Example:               safeDrainExample,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(o.Complete(f, cmd, args))
		},
	}

	cmdutil.AddDryRunFlag(cmd)
	return cmd
}

func (o *SafeDrainCmdOptions) Complete(f cmdutil.Factory, cmd *cobra.Command, args []string) error {
	return nil
}
