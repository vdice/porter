package porter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/deislabs/cnab-go/action"
	"github.com/deislabs/cnab-go/driver"
	"github.com/pkg/errors"

	"github.com/deislabs/porter/pkg/config"
)

// InstallOptions that may be specified when installing a bundle.
// Porter handles defaulting any missing values.
type InstallOptions struct {
	BundleLifecycleOpts
}

// InstallBundle accepts a set of pre-validated InstallOptions and uses
// them to install a bundle.
func (p *Porter) InstallBundle(opts InstallOptions) error {
	err := p.prepullBundleByTag(&opts.BundleLifecycleOpts)
	if err != nil {
		return errors.Wrap(err, "unable to pull bundle before installation")
	}

	err = p.applyDefaultOptions(&opts.sharedOptions)
	if err != nil {
		return err
	}

	err = p.ensureLocalBundleIsUpToDate(opts.bundleFileOptions)
	if err != nil {
		return err
	}

	deperator := newDependencyExecutioner(p)
	err = deperator.Prepare(opts.BundleLifecycleOpts, p.CNAB.Install)
	if err != nil {
		return err
	}

	err = deperator.Execute(config.ActionInstall)
	if err != nil {
		return err
	}

	opRelocator, err := makeOpRelocator(opts.RelocationMapping)
	if err != nil {
		return err
	}
	actionArgs := opts.ToActionArgs(deperator)
	actionArgs.OperationConfigs = append(actionArgs.OperationConfigs, opRelocator)

	fmt.Fprintf(p.Out, "installing %s...\n", opts.Name)
	return p.CNAB.Install(actionArgs)
}

// From Duffle
func makeOpRelocator(relMapping string) (action.OperationConfigFunc, error) {
	rm, err := loadRelMapping(relMapping)
	if err != nil {
		return nil, err
	}

	relMap := make(map[string]string)
	if rm != "" {
		err := json.Unmarshal([]byte(rm), &relMap)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal relocation mapping: %v", err)
		}
	}

	return func(op *driver.Operation) error {
		// if there is a relocation mapping, ensure it is mounted and relocate the invocation image
		if rm != "" {
			op.Files["/cnab/app/relocation-mapping.json"] = rm

			im, ok := relMap[op.Image.Image]
			if !ok {
				return fmt.Errorf("invocation image %s not present in relocation mapping %v", op.Image.Image, relMap)
			}
			op.Image.Image = im
		}
		return nil
	}, nil
}

// From Duffle
func loadRelMapping(relMap string) (string, error) {
	if relMap != "" {
		data, err := ioutil.ReadFile(relMap)
		if err != nil {
			return "", fmt.Errorf("failed to read relocation mapping from %s: %v", relMap, err)
		}
		return string(data), nil
	}

	return "", nil
}
