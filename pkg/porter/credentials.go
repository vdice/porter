package porter

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/deislabs/porter/pkg/context"
	"github.com/deislabs/porter/pkg/credentialsgenerator"
	"github.com/deislabs/porter/pkg/printer"

	dtprinter "github.com/carolynvs/datetime-printer"
	credentials "github.com/deislabs/cnab-go/credentials"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// CredentialShowOptions represent options for Porter's credential show command
type CredentialShowOptions struct {
	RawFormat string
	Format    printer.Format
	Name      string
}

// CredentialsFile represents a CNAB credentials file and corresponding metadata
type CredentialsFile struct {
	Name     string
	Modified time.Time
}

// CredentialsFileList is a slice of CredentialsFiles
type CredentialsFileList []CredentialsFile

func (l CredentialsFileList) Len() int {
	return len(l)
}
func (l CredentialsFileList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}
func (l CredentialsFileList) Less(i, j int) bool {
	return l[i].Modified.Before(l[j].Modified)
}

// fetchCredentials fetches all credentials from the designated credentials dir
func (p *Porter) fetchCredentials() (*CredentialsFileList, error) {
	credsDir, err := p.Config.GetCredentialsDir()
	if err != nil {
		return &CredentialsFileList{}, errors.Wrap(err, "unable to determine credentials directory")
	}

	credentialsFiles := CredentialsFileList{}
	if ok, _ := p.Context.FileSystem.DirExists(credsDir); ok {
		p.Context.FileSystem.Walk(credsDir, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				credSet := &credentials.CredentialSet{}
				data, err := p.Context.FileSystem.ReadFile(path)
				if err != nil {
					if p.Debug {
						fmt.Fprintf(p.Err, "unable to load credential set from %s: %s\n", path, err)
					}
					return nil
				}
				if err = yaml.Unmarshal(data, credSet); err != nil {
					if p.Debug {
						fmt.Fprintf(p.Err, "unable to unmarshal credential set from file %s: %s\n", info.Name(), err)
					}
					return nil
				}
				credentialsFiles = append(credentialsFiles,
					CredentialsFile{Name: credSet.Name, Modified: info.ModTime()})
			}
			return nil
		})
		sort.Sort(sort.Reverse(credentialsFiles))
	}
	return &credentialsFiles, nil
}

// ListCredentials lists credentials using the provided printer.PrintOptions
func (p *Porter) ListCredentials(opts printer.PrintOptions) error {
	credentialsFiles, err := p.fetchCredentials()
	if err != nil {
		return errors.Wrap(err, "unable to fetch credentials")
	}

	switch opts.Format {
	case printer.FormatJson:
		return printer.PrintJson(p.Out, *credentialsFiles)
	case printer.FormatYaml:
		return printer.PrintYaml(p.Out, *credentialsFiles)
	case printer.FormatTable:
		// have every row use the same "now" starting ... NOW!
		now := time.Now()
		tp := dtprinter.DateTimePrinter{
			Now: func() time.Time { return now },
		}

		printCredRow :=
			func(v interface{}) []string {
				cr, ok := v.(CredentialsFile)
				if !ok {
					return nil
				}
				return []string{cr.Name, tp.Format(cr.Modified)}
			}
		return printer.PrintTable(p.Out, *credentialsFiles, printCredRow,
			"NAME", "MODIFIED")
	default:
		return fmt.Errorf("invalid format: %s", opts.Format)
	}
}

type CredentialOptions struct {
	sharedOptions
	DryRun bool
	Silent bool
}

// Validate prepares for an action and validates the options.
// For example, relative paths are converted to full paths and then checked that
// they exist and are accessible.
func (g *CredentialOptions) Validate(args []string, cxt *context.Context) error {
	err := g.validateCredName(args)
	if err != nil {
		return err
	}

	err = g.validateBundlePath(cxt)
	if err != nil {
		return err
	}
	return nil
}

func (g *CredentialOptions) validateCredName(args []string) error {
	if len(args) == 1 {
		g.Name = args[0]
	} else if len(args) > 1 {
		return errors.Errorf("only one positional argument may be specified, the credential name, but multiple were received: %s", args)
	}
	return nil
}

// GenerateCredentials builds a new credential set based on the given options. This can be either
// a silent build, based on the opts.Silent flag, or interactive using a survey. Returns an
// error if unable to generate credentials
func (p *Porter) GenerateCredentials(opts CredentialOptions) error {

	//TODO make this work for either porter.yaml OR a bundle
	bundle, err := p.CNAB.LoadBundle(opts.File, opts.Insecure)
	if err != nil {
		return err
	}
	name := opts.Name
	if name == "" {
		name = bundle.Name
	}
	genOpts := credentialsgenerator.GenerateOptions{
		Name:        name,
		Credentials: bundle.Credentials,
		Silent:      opts.Silent,
	}
	fmt.Fprintf(p.Out, "Generating new credential %s from bundle %s\n", genOpts.Name, bundle.Name)
	fmt.Fprintf(p.Out, "==> %d credentials required for bundle %s\n", len(genOpts.Credentials), bundle.Name)

	cs, err := credentialsgenerator.GenerateCredentials(genOpts)
	if err != nil {
		return errors.Wrap(err, "unable to generate credentials")
	}

	//write the credential out to PORTER_HOME with Porter's Context
	data, err := yaml.Marshal(cs)
	if err != nil {
		return errors.Wrap(err, "unable to generate credentials YAML")
	}
	if opts.DryRun {
		fmt.Fprintf(p.Out, "%v", string(data))
		return nil
	}
	credentialsDir, err := p.Config.GetCredentialsDir()
	if err != nil {
		return errors.Wrap(err, "unable to get credentials directory")
	}
	// Make the credentials path if it doesn't exist. MkdirAll does nothing if it already exists
	// Readable, writable only by the user
	err = p.Config.FileSystem.MkdirAll(credentialsDir, 0700)
	if err != nil {
		return errors.Wrap(err, "unable to create credentials directory")
	}
	dest, err := p.Config.GetCredentialPath(genOpts.Name)
	if err != nil {
		return errors.Wrap(err, "unable to determine credentials path")
	}

	fmt.Fprintf(p.Out, "Saving credential to %s\n", dest)

	err = p.Context.FileSystem.WriteFile(dest, data, 0600)
	if err != nil {
		return errors.Wrapf(err, "couldn't write credential file %s", dest)
	}
	return nil
}

// Validate validates the args provided Porter's credential show command
func (o *CredentialShowOptions) Validate(args []string) error {
	switch len(args) {
	case 0:
		return errors.Errorf("no credential name was specified")
	case 1:
		o.Name = strings.ToLower(args[0])
	default:
		return errors.Errorf("only one positional argument may be specified, the credential name, but multiple were received: %s", args)
	}

	format, err := printer.ParseFormat(o.RawFormat)
	if err != nil {
		return err
	}
	o.Format = format

	return nil
}

// ShowCredential shows the credential set corresponding to the provided name, using
// the provided printer.PrintOptions for display.
func (p *Porter) ShowCredential(opts CredentialShowOptions) error {
	credsDir, err := p.Config.GetCredentialsDir()
	if err != nil {
		return errors.Wrap(err, "unable to determine credentials directory")
	}

	credSet := &credentials.CredentialSet{}
	data, err := p.Context.FileSystem.ReadFile(filepath.Join(credsDir, fmt.Sprintf("%s.yaml", opts.Name)))
	if err != nil {
		return errors.Wrapf(err, "unable to load credential set %s", opts.Name)
	}

	if err = yaml.Unmarshal(data, credSet); err != nil {
		return errors.Wrapf(err, "unable to unmarshal credential set %s", opts.Name)
	}

	switch opts.Format {
	case printer.FormatJson:
		return printer.PrintJson(p.Out, credSet)
	case printer.FormatYaml:
		return printer.PrintYaml(p.Out, credSet)
	case printer.FormatTable:
		printCredentialRow :=
			func(v interface{}) []string {
				cs, ok := v.(credentials.CredentialStrategy)
				if !ok {
					return nil
				}

				// Build a reflected Source of type reflectedStruct, for use below
				reflectedSource := reflectedStruct{
					Value: reflect.ValueOf(cs.Source),
					Type:  reflect.TypeOf(cs.Source),
				}

				// Determine the source type by seeing which reflected source's field
				// corresponds to a non-empty reflected source value
				var source string
				var sourceType string
				// Iterate through all of the fields of a credentials.Source struct
				for i := 0; i < reflectedSource.Type.NumField(); i++ {
					// A Field name would be 'Path', 'EnvVar', etc.
					fieldName := reflectedSource.Type.Field(i).Name
					// Get the value for said Field
					fieldValue := reflect.Indirect(reflectedSource.Value).FieldByName(fieldName).String()
					// If not empty, this field value and name represent our source and source type, respectively
					if fieldValue != "" {
						source = fieldValue
						sourceType = fieldName
					}
				}
				return []string{cs.Name, source, sourceType}
			}
		return printer.PrintTable(p.Out, credSet.Credentials, printCredentialRow,
			"Name", "Local Source", "Source Type")
	default:
		return fmt.Errorf("invalid format: %s", opts.Format)
	}
}

type reflectedStruct struct {
	Value reflect.Value
	Type  reflect.Type
}
