package porter

import (
	"get.porter.sh/porter/pkg/manifest"
	"github.com/cnabio/cnab-go/bundle"
	"github.com/pkg/errors"
)

// applyDefaultOptions applies more advanced defaults to the options
// based on values that beyond just what was supplied by the user
// such as information in the manifest itself.
func (p *Porter) applyDefaultOptions(opts *sharedOptions) error {
	if opts.File != "" {
		err := p.LoadManifestFrom(opts.File)
		if err != nil {
			return err
		}
	}

	// Ensure that we have a manifest initialized, even if it's just an empty one
	// This happens for non-porter bundles using --cnab-file or --tag
	if p.Manifest == nil {
		p.Manifest = &manifest.Manifest{}
	}

	//
	// Default the claim name to the bundle name
	//
	if opts.Name == "" {
		if p.Manifest.Name != "" {
			opts.Name = p.Manifest.Name
		} else if opts.CNABFile != "" {
			name, err := p.getCNABFileName(opts.CNABFile)
			if err != nil {
				return err
			}
			opts.Name = name
		}
	}

	//
	// Default the porter-debug param to --debug
	//
	if _, set := opts.combinedParameters["porter-debug"]; !set && p.Debug {
		if opts.combinedParameters == nil {
			opts.combinedParameters = make(map[string]string)
		}
		opts.combinedParameters["porter-debug"] = "true"
	}

	return nil
}

func (p *Porter) getCNABFileName(filepath string) (string, error) {
	data, err := p.FileSystem.ReadFile(filepath)
	if err != nil {
		return "", errors.Wrapf(err, "unable to read cnab file %s", filepath)
	}

	bun, err := bundle.Unmarshal(data)
	if err != nil {
		return "", errors.Wrapf(err, "unable to parse cnab file %s", filepath)
	}

	return bun.Name, nil
}
