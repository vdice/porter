package porter

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/deislabs/porter/pkg/config"
)

// UninstallOptions that may be specified when uninstalling a bundle.
// Porter handles defaulting any missing values.
type UninstallOptions struct {
	BundleLifecycleOpts
}

// UninstallBundle accepts a set of pre-validated UninstallOptions and uses
// them to uninstall a bundle.
func (p *Porter) UninstallBundle(opts UninstallOptions) error {
	err := p.prepullBundleByTag(&opts.BundleLifecycleOpts)
	if err != nil {
		return errors.Wrap(err, "unable to pull bundle before uninstall")
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
	err = deperator.Prepare(opts.BundleLifecycleOpts, p.CNAB.Uninstall)
	if err != nil {
		return err
	}

	opRelocator, err := makeOpRelocator(opts.RelocationMapping)
	if err != nil {
		return err
	}
	actionArgs := opts.ToActionArgs(deperator)
	actionArgs.OperationConfigs = append(actionArgs.OperationConfigs, opRelocator)

	fmt.Fprintf(p.Out, "uninstalling %s...\n", opts.Name)
	err = p.CNAB.Uninstall(actionArgs)
	if err != nil {
		if len(deperator.deps) > 0 {
			return errors.Wrapf(err, "failed to uninstall the %s bundle, the remaining dependencies were not uninstalled", opts.Name)
		} else {
			return err
		}
	}

	// TODO: See https://github.com/deislabs/porter/issues/465 for flag to allow keeping around the dependencies
	return deperator.Execute(config.ActionUninstall)
}
