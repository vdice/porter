package main

import (
	"github.com/deislabs/porter/pkg/porter"
	"github.com/deislabs/porter/pkg/printer"
	"github.com/spf13/cobra"
)

func buildCredentialsCommand(p *porter.Porter) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "credentials",
		Aliases:     []string{"credential", "cred", "creds"},
		Annotations: map[string]string{"group": "resource"},
		Short:       "Credentials commands",
	}

	cmd.AddCommand(buildCredentialsAddCommand(p))
	cmd.AddCommand(buildCredentialsEditCommand(p))
	cmd.AddCommand(buildCredentialsGenerateCommand(p))
	cmd.AddCommand(buildCredentialsListCommand(p))
	cmd.AddCommand(buildCredentialsRemoveCommand(p))
	cmd.AddCommand(buildCredentialsShowCommand(p))

	return cmd
}

func buildCredentialsAddCommand(p *porter.Porter) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "add",
		Short:  "Add Credential",
		Hidden: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			p.PrintVersion()
			return nil
		},
	}
	return cmd
}

func buildCredentialsEditCommand(p *porter.Porter) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "edit",
		Short:  "Edit Credential",
		Hidden: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			p.PrintVersion()
			return nil
		},
	}
	return cmd
}

func buildCredentialsGenerateCommand(p *porter.Porter) *cobra.Command {
	opts := porter.CredentialOptions{}
	cmd := &cobra.Command{
		Use:   "generate [NAME]",
		Short: "Generate Credential Set",
		Long: `Generate a named set of credentials.

The first argument is the name of credential set you wish to generate. If not
provided, this will default to the bundle name. By default, Porter will 
generate a credential set for the bundle in the current directory. You may also
specify a bundle with --file.

Bundles define 1 or more credential(s) that are required to interact with a 
bundle. The bundle definition defines where the credential should be delivered
to the bundle, i.e. at /root/.kube. A credential set, on the other hand, 
represents the source data that you wish to use when interacting with the 
bundle. These will typically be environment variables or files on your local 
file system. 

When you wish to install, upgrade or delete a bundle, Porter will use the 
credential set to determine where to read the necessary information from and
will then provide it to the bundle in the correct location. `,
		Example: `  porter credential generate
  porter bundle credential generate kubecred --insecure
  porter bundle credential generate kubecred --file myapp/bundle.json
  porter bundle credential generate kubecred --file myapp/bundle.json --dry-run
`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.Validate(args, p.Context)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return p.GenerateCredentials(opts)
		},
	}

	f := cmd.Flags()
	f.BoolVar(&opts.Insecure, "insecure", true,
		"Allow working with untrusted bundles.")
	f.StringVarP(&opts.File, "file", "f", "",
		"Path to the CNAB definition to install. Defaults to the bundle in the current directory.")
	f.BoolVar(&opts.DryRun, "dry-run", false,
		"Generate credential but do not save it.")
	return cmd
}

func buildCredentialsListCommand(p *porter.Porter) *cobra.Command {
	opts := porter.ListOptions{}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List credentials",
		Long:    `List named sets of credentials defined by the user.`,
		Example: `  porter credentials list [-o table|json|yaml]`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			opts.Format, err = printer.ParseFormat(opts.RawFormat)
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return p.ListCredentials(printer.PrintOptions{Format: opts.Format})
		},
	}

	f := cmd.Flags()
	f.StringVarP(&opts.RawFormat, "output", "o", "table",
		"Specify an output format.  Allowed values: table, json, yaml")

	return cmd
}

func buildCredentialsRemoveCommand(p *porter.Porter) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "remove",
		Short:  "Remove a Credential",
		Hidden: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			p.PrintVersion()
			return nil
		},
	}
	return cmd
}

func buildCredentialsShowCommand(p *porter.Porter) *cobra.Command {
	opts := porter.CredentialShowOptions{}

	cmd := &cobra.Command{
		Use:     "show",
		Short:   "Show a Credential",
		Long:    `Show a particular credential set, including all named credentials and their corresponding mappings.`,
		Example: `  porter credential show NAME [-o table|json|yaml]`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.Validate(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return p.ShowCredential(opts)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&opts.RawFormat, "output", "o", "table",
		"Specify an output format.  Allowed values: table, json, yaml")

	return cmd
}
