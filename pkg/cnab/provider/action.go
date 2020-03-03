package cnabprovider

import (
	"encoding/json"

	"get.porter.sh/porter/pkg/config"
	"github.com/cnabio/cnab-go/action"
	"github.com/cnabio/cnab-go/driver"
	"github.com/docker/cnab-to-oci/relocation"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
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

	// Params is the set of parameters to pass to the bundle.
	Params map[string]string

	// Either a filepath to a credential file or the name of a set of a credentials.
	CredentialIdentifiers []string

	// Driver is the CNAB-compliant driver used to run bundle actions.
	Driver string

	// Path to an optional relocation mapping file
	RelocationMapping string
}

func (d *Runtime) ApplyConfig(args ActionArguments) action.OperationConfigs {
	return action.OperationConfigs{
		d.SetOutput(),
		d.AddFiles(args),
		d.AddRelocation(args),
	}
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

// AddRelocation operates on an ActionArguments and adds any provided relocation mapping
// to the operation's files.
func (d *Runtime) AddRelocation(args ActionArguments) action.OperationConfigFunc {
	return func(op *driver.Operation) error {
		if args.RelocationMapping != "" {
			b, err := d.FileSystem.ReadFile(args.RelocationMapping)
			if err != nil {
				return errors.Wrap(err, "unable to add relocation mapping")
			}
			op.Files["/cnab/app/relocation-mapping.json"] = string(b)
			var reloMap relocation.ImageRelocationMap
			err = json.Unmarshal(b, &reloMap)
			// If the invocation image is present in the relocation mapping, we need
			// to update the operation and set the new image reference. Unfortunately,
			// the relocation mapping is just reference => reference, so there isn't a
			// great way to check for the invocation image.
			if mappedInvo, ok := reloMap[op.Image.Image]; ok {
				op.Image.Image = mappedInvo
			}
		}
		return nil
	}
}
