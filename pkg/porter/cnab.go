package porter

import (
	"fmt"
	"os"
	"path/filepath"

	"get.porter.sh/porter/pkg/build"
	cnabprovider "get.porter.sh/porter/pkg/cnab/provider"
	"get.porter.sh/porter/pkg/config"
	"get.porter.sh/porter/pkg/context"
	"get.porter.sh/porter/pkg/parameters"
	"github.com/cnabio/cnab-go/driver/command"
	"github.com/pkg/errors"
)

const (
	// DockerDriver is the name of the Docker driver.
	DockerDriver = cnabprovider.DriverNameDocker

	// DebugDriver is the name of the Debug driver.
	DebugDriver = cnabprovider.DriverNameDebug

	// DefaultDriver is the name of the default driver (Docker).
	DefaultDriver = DockerDriver
)

type bundleFileOptions struct {
	// File path to the porter manifest. Defaults to the bundle in the current directory.
	File string

	// CNABFile is the path to the bundle.json file. Cannot be specified at the same time as the porter manifest or a tag.
	CNABFile string

	// RelocationMapping is the path to the relocation-mapping.json file, if one exists. Populated only for published bundles
	RelocationMapping string
}

func (o *bundleFileOptions) Validate(cxt *context.Context) error {
	err := o.validateBundleFiles(cxt)
	if err != nil {
		return err
	}

	err = o.defaultBundleFiles(cxt)
	if err != nil {
		return err
	}

	return err
}

// SharedOptions are common options that apply to multiple CNAB actions.
type SharedOptions struct {
	bundleFileOptions

	// Name of the instance. Defaults to the name of the bundle.
	Name string

	// Params is the unparsed list of NAME=VALUE parameters set on the command line.
	Params []string

	// ParameterSets is a list of parameter sets containing parameter sources
	ParameterSets []string

	// CredentialIdentifiers is a list of credential names or paths to make available to the bundle.
	CredentialIdentifiers []string

	// Driver is the CNAB-compliant driver used to run bundle actions.
	Driver string

	// parsedParams is the parsed set of parameters from Params.
	parsedParams map[string]string
}

// Validate prepares for an action and validates the options.
// For example, relative paths are converted to full paths and then checked that
// they exist and are accessible.
func (o *SharedOptions) Validate(args []string, cxt *context.Context) error {
	err := o.validateInstanceName(args)
	if err != nil {
		return err
	}

	err = o.bundleFileOptions.Validate(cxt)
	if err != nil {
		return err
	}

	err = o.validateParams(cxt)
	if err != nil {
		return err
	}

	o.defaultDriver()
	err = o.validateDriver()
	if err != nil {
		return err
	}

	return nil
}

// validateInstanceName grabs the claim name from the first positional argument.
func (o *SharedOptions) validateInstanceName(args []string) error {
	if len(args) == 1 {
		o.Name = args[0]
	} else if len(args) > 1 {
		return errors.Errorf("only one positional argument may be specified, the bundle instance name, but multiple were received: %s", args)
	}

	return nil
}

// defaultBundleFiles defaults the porter manifest and the bundle.json files.
func (o *bundleFileOptions) defaultBundleFiles(cxt *context.Context) error {
	if o.File == "" {
		pwd, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "could not get current working directory")
		}

		manifestExists, err := cxt.FileSystem.Exists(filepath.Join(pwd, config.Name))
		if err != nil {
			return errors.Wrap(err, "could not check if porter manifest exists in current directory")
		}

		if manifestExists {
			o.File = config.Name
			o.CNABFile = build.LOCAL_BUNDLE
		}
	} else {
		bundleDir := filepath.Dir(o.File)
		o.CNABFile = filepath.Join(bundleDir, build.LOCAL_BUNDLE)
	}

	return nil
}

func (o *bundleFileOptions) validateBundleFiles(cxt *context.Context) error {
	if o.File != "" && o.CNABFile != "" {
		return errors.New("cannot specify both --file and --cnab-file")
	}

	err := o.validateFile(cxt)
	if err != nil {
		return err
	}

	err = o.validateCNABFile(cxt)
	if err != nil {
		return err
	}

	return nil
}

func (o *bundleFileOptions) validateFile(cxt *context.Context) error {
	if o.File == "" {
		return nil
	}

	// Verify the file can be accessed
	if _, err := cxt.FileSystem.Stat(o.File); err != nil {
		return errors.Wrapf(err, "unable to access --file %s", o.File)
	}

	return nil
}

// validateCNABFile converts the bundle file path to an absolute filepath and verifies that it exists.
func (o *bundleFileOptions) validateCNABFile(cxt *context.Context) error {
	if o.CNABFile == "" {
		return nil
	}

	originalPath := o.CNABFile
	if !filepath.IsAbs(o.CNABFile) {
		// Convert to an absolute filepath because runtime needs it that way
		pwd, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "could not get current working directory")
		}

		f := filepath.Join(pwd, o.CNABFile)
		f, err = filepath.Abs(f)
		if err != nil {
			return errors.Wrapf(err, "could not get absolute path for --cnab-file %s", o.CNABFile)
		}

		o.CNABFile = f
	}

	// Verify the file can be accessed
	if _, err := cxt.FileSystem.Stat(o.CNABFile); err != nil {
		// warn about the original relative path
		return errors.Wrapf(err, "unable to access --cnab-file %s", originalPath)
	}

	return nil
}

func (o *SharedOptions) validateParams(cxt *context.Context) error {
	err := o.parseParams()
	if err != nil {
		return err
	}

	return nil
}

// parsedParams parses the variable assignments in Params.
func (o *SharedOptions) parseParams() error {
	p, err := parameters.ParseVariableAssignments(o.Params)
	if err != nil {
		return err
	}
	o.parsedParams = p
	return nil
}

// defaultDriver supplies the default driver if none is specified
func (o *SharedOptions) defaultDriver() {
	if o.Driver == "" {
		o.Driver = DefaultDriver
	}
}

// validateDriver validates that the provided driver is supported by Porter
func (o *SharedOptions) validateDriver() error {
	switch o.Driver {
	case DockerDriver, DebugDriver:
		return nil
	default:
		cmddriver := &command.Driver{Name: o.Driver}
		if cmddriver.CheckDriverExists() {
			return nil
		}

		return fmt.Errorf("unsupported driver or driver not found in PATH: %s", o.Driver)
	}
}
