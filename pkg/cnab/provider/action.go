package cnabprovider

import (
	"github.com/deislabs/cnab-go/action"
	"github.com/deislabs/cnab-go/driver"
)

// Shared arguments for all CNAB actions
type ActionArguments struct {
	// Name of the instance.
	Claim string

	// Either a filepath to the bundle or the name of the bundle.
	BundlePath string

	// Additional files to copy into the bundle
	// Target Path => File Contents
	Files map[string]string

	// Insecure bundle action allowed.
	Insecure bool

	// Params is the set of parameters to pass to the bundle.
	Params map[string]string

	// Either a filepath to a credential file or the name of a set of a credentials.
	CredentialIdentifiers []string

	// Driver is the CNAB-compliant driver used to run bundle actions.
	Driver string

	// OperationConfigs is the set of action.OperationConfigs that can be applied to the operation
	OperationConfigs action.OperationConfigs
}

func (d *Runtime) ApplyDefaultConfig(args ActionArguments) action.OperationConfigs {
	opConfigs := args.OperationConfigs
	opConfigs = append(opConfigs, d.SetOutput())
	opConfigs = append(opConfigs, d.AddFiles(args))

	return opConfigs
}

func (d *Runtime) SetOutput() action.OperationConfigFunc {
	return func(op *driver.Operation) error {
		op.Out = d.Out
		return nil
	}
}

func (d *Runtime) AddFiles(args ActionArguments) action.OperationConfigFunc {
	return func(op *driver.Operation) error {
		for k, v := range args.Files {
			op.Files[k] = v
		}
		return nil
	}
}
