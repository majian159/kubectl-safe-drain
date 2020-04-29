package sdrain

import (
	"fmt"
	"kubectl-sdrain/pkg/sdrain"

	"k8s.io/kubectl/pkg/cmd/drain"

	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"k8s.io/cli-runtime/pkg/resource"

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
	nodeInfos   []*resource.Info

	genericclioptions.IOStreams
}

var (
	safeDrainLong = templates.LongDesc(i18n.T(`
		Safe drain node in preparation for maintenance.
		The given node will be marked unschedulable to prevent new pods from arriving.`))

	safeDrainExample = templates.Examples(i18n.T(`
		# Safe drain node "foo", even if there are pods not managed by a ReplicationController, ReplicaSet, Job or StatefulSet on it.
		$ kubectl safe-drain foo`))
)

func NewDrainCmdOptions(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) *SafeDrainCmdOptions {
	o := &SafeDrainCmdOptions{
		PrintFlags: genericclioptions.NewPrintFlags("safe drained").WithTypeSetter(scheme.Scheme),
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
		Use:                   "safe-drain NODE",
		DisableFlagsInUseLine: true,
		Short:                 i18n.T("Safe drain node in preparation for maintenance"),
		Long:                  safeDrainLong,
		Example:               safeDrainExample,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(o.Complete(f, cmd, args))
			cmdutil.CheckErr(o.RunCordon(f))
			cmdutil.CheckErr(o.RunSafeDrain())
		},
	}

	cmd.Flags().BoolVar(&o.safeDrainer.Force, "force", o.safeDrainer.Force, "Continue even if there are pods not managed by a ReplicationController, ReplicaSet, Job, DaemonSet or StatefulSet.")
	cmd.Flags().BoolVar(&o.safeDrainer.DeleteLocalData, "delete-local-data", o.safeDrainer.DeleteLocalData, "Continue even if there are pods using emptyDir (local data that will be deleted when the node is drained).")
	cmd.Flags().DurationVar(&o.safeDrainer.Timeout, "timeout", o.safeDrainer.Timeout, "The length of time to wait before giving up, zero means infinite")
	cmd.Flags().StringVarP(&o.safeDrainer.Selector, "selector", "l", o.safeDrainer.Selector, "Selector (label query) to filter on")

	cmdutil.AddDryRunFlag(cmd)
	return cmd
}

func (o *SafeDrainCmdOptions) Complete(f cmdutil.Factory, cmd *cobra.Command, args []string) error {
	var err error

	if len(args) == 0 && !cmd.Flags().Changed("selector") {
		return cmdutil.UsageErrorf(cmd, fmt.Sprintf("USAGE: %s [flags]", cmd.Use))
	}
	if len(args) > 0 && len(o.safeDrainer.Selector) > 0 {
		return cmdutil.UsageErrorf(cmd, "error: cannot specify both a node name and a --selector option")
	}

	o.safeDrainer.DryRun = cmdutil.GetDryRunFlag(cmd)

	if o.safeDrainer.Client, err = f.KubernetesClientSet(); err != nil {
		return err
	}

	o.ToPrinter = func(operation string) (printers.ResourcePrinterFunc, error) {
		o.PrintFlags.NamePrintFlags.Operation = operation
		if o.safeDrainer.DryRun {
			err := o.PrintFlags.Complete("%s (dry run)")
			if err != nil {
				return nil, err
			}
		}

		printer, err := o.PrintFlags.ToPrinter()
		if err != nil {
			return nil, err
		}

		return printer.PrintObj, nil
	}

	builder := f.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		ResourceNames("nodes", args...).
		SingleResourceType().
		Flatten()

	if len(o.safeDrainer.Selector) > 0 {
		builder = builder.LabelSelectorParam(o.safeDrainer.Selector).
			ResourceTypes("nodes")
	}

	r := builder.Do()

	if err = r.Err(); err != nil {
		return err
	}

	return r.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		if info.Mapping.Resource.GroupResource() != (schema.GroupResource{Group: "", Resource: "nodes"}) {
			return fmt.Errorf("error: expected resource of type node, got %q", info.Mapping.Resource)
		}

		o.nodeInfos = append(o.nodeInfos, info)
		return nil
	})
}

func (o *SafeDrainCmdOptions) RunCordon(f cmdutil.Factory) error {
	var args []string
	for _, info := range o.nodeInfos {
		args = append(args, info.Name)
	}

	if o.safeDrainer.DryRun {
		args = append(args, "--dry-run")
	}

	cordonCmd := drain.NewCmdCordon(f, o.IOStreams)
	cordonCmd.SetArgs(args)
	return cordonCmd.Execute()
}

func (o *SafeDrainCmdOptions) RunSafeDrain() error {
	printObj, err := o.ToPrinter("safe drained")
	if err != nil {
		return err
	}

	drainedNodes := sets.NewString()

	var fatal error
	for _, info := range o.nodeInfos {
		var err error

		if !o.safeDrainer.DryRun {
			err = o.safeDeleteOrEvictPodsSimple(info)
		}

		if err == nil || o.safeDrainer.DryRun {
			drainedNodes.Insert(info.Name)
			printObj(info.Object, o.Out)
			continue
		} else {
			fmt.Fprintf(o.ErrOut, "error: unable to safe-drain node %q, aborting command...\n\n", info.Name)
			var remainingNodes []string
			fatal = err
			for _, remainingInfo := range o.nodeInfos {
				if drainedNodes.Has(remainingInfo.Name) {
					continue
				}
				remainingNodes = append(remainingNodes, remainingInfo.Name)
			}

			if len(remainingNodes) > 0 {
				fmt.Fprintf(o.ErrOut, "There are pending nodes to be drained:\n")
				for _, nodeName := range remainingNodes {
					fmt.Fprintf(o.ErrOut, " %s\n", nodeName)
				}
			}
			break
		}

	}

	return fatal
}

func (o *SafeDrainCmdOptions) safeDeleteOrEvictPodsSimple(nodeInfo *resource.Info) error {
	list, errs := o.safeDrainer.GetPodsForDeletion(nodeInfo.Name)
	if errs != nil {
		return utilerrors.NewAggregate(errs)
	}

	if err := o.safeDrainer.MigratePods(list); err != nil {
		pendingList, newErrs := o.safeDrainer.GetPodsForDeletion(nodeInfo.Name)
		if pendingList != nil {
			pods := pendingList.Pods()
			if len(pods) != 0 {
				fmt.Fprintf(o.ErrOut, "There are pending pods in node %q when an error occurred: %v\n", nodeInfo.Name, err)
				for _, pendingPod := range pods {
					fmt.Fprintf(o.ErrOut, "%s/%s\n", "pod", pendingPod.Name)
				}
			}
		}
		if newErrs != nil {
			fmt.Fprintf(o.ErrOut, "Following errors occurred while getting the list of pods to delete:\n%s", utilerrors.NewAggregate(newErrs))
		}
		return err
	}
	return nil
}
