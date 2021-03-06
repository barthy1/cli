// Copyright © 2019 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eventlistener

import (
	"errors"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/printer"
	"github.com/tektoncd/cli/pkg/validate"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	emptyMsg = "No eventlisteners found"
)

func listCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("list")

	eg := `List all eventlisteners in namespace 'bar':

	tkn eventlistener list -n bar

or

	tkn el ls -n bar
`

	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lists eventlisteners in a namespace",
		Annotations: map[string]string{
			"commandType": "main",
		},
		Example: eg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.NamespaceExists(p); err != nil {
				return err
			}

			cs, err := p.Clients()
			if err != nil {
				return err
			}

			els, err := list(cs.Triggers, p.Namespace())
			if err != nil {
				return fmt.Errorf(`failed to list eventlisteners from %s namespace \n`, p.Namespace())
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return errors.New(`output option not set properly \n`)
			}

			stream := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if output != "" {
				return printer.PrintObject(stream.Out, els, f)
			}

			if err = printFormatted(stream, els, p); err != nil {
				return errors.New(`failed to print eventlisteners \n`)
			}
			return nil
		},
	}

	f.AddFlags(c)
	return c
}

func list(client versioned.Interface, namespace string) (*v1alpha1.EventListenerList, error) {
	els, err := client.TektonV1alpha1().EventListeners(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// NOTE: this is required for -o json|yaml to work properly since
	// tektoncd go client fails to set these; probably a bug
	els.GetObjectKind().SetGroupVersionKind(
		schema.GroupVersionKind{
			Version: "tekton.dev/v1alpha1",
			Kind:    "EventListenerList",
		})

	return els, nil
}

func printFormatted(s *cli.Stream, els *v1alpha1.EventListenerList, p cli.Params) error {
	if len(els.Items) == 0 {
		fmt.Fprintln(s.Err, emptyMsg)
		return nil
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, "NAME\tAGE")
	for _, el := range els.Items {
		fmt.Fprintf(w, "%s\t%s\n",
			el.Name,
			formatted.Age(&el.CreationTimestamp, p.Time()),
		)
	}

	return w.Flush()
}
