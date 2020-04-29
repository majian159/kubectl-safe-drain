package main

import (
	"kubectl-sdrain/cmd/sdrain"
	"os"

	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func main() {
	flags := pflag.NewFlagSet("kubectl-sdrain", pflag.ExitOnError)
	pflag.CommandLine = flags

	cf := genericclioptions.NewConfigFlags(true)
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(cf)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	root := sdrain.NewCmdSafeDrain(f, genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
