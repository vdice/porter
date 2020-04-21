package linter

import (
	"encoding/json"
	"fmt"
	"strings"

	"get.porter.sh/porter/pkg/cnab/extensions"
	"get.porter.sh/porter/pkg/context"
	"get.porter.sh/porter/pkg/manifest"
	"get.porter.sh/porter/pkg/mixin/query"
	"get.porter.sh/porter/pkg/pkgmgmt"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
)

// Level of severity for a lint result.
type Level int

func (l Level) String() string {
	switch l {
	case LevelError:
		return "error"
	case LevelWarning:
		return "warning"
	}
	return ""
}

// Code representing the problem identified by the linter
// Recommended to use the pattern MIXIN-NUMBER so that you don't collide with
// codes from another mixin or with Porter's codes.
// Example:
// - exec-105
// - helm-410
type Code string

const (
	// LevelError indicates a lint result is an error that will prevent the bundle from building properly.
	LevelError Level = 0

	// LevelWarning indicates a lint result is a warning about a best practice or identifies a problem that is not
	// guaranteed to break the build.
	LevelWarning Level = 2
)

// Result is a single item identified by the linter.
type Result struct {
	// Level of severity
	Level Level

	// Location of the problem in the manifest.
	Location Location

	// Code uniquely identifying the type of problem.
	Code Code

	// Title to display (80 chars).
	Title string

	// Message explaining the problem.
	Message string

	// URL that provides additional assistance with this problem.
	URL string
}

// Location represents generic location data
type Location struct {
	Data interface{}
}

// UnmarshalJSON unmarshals Location data into the appropriate location type
func (u *Location) UnmarshalJSON(data []byte) error {
	// Try unmarshaling to the supported Location types

	// ActionLocation
	al := map[string]ActionLocation{}
	err := json.Unmarshal(data, &al)
	if _, ok := err.(*json.UnmarshalTypeError); err != nil && !ok {
		return err
	}
	if al["Data"].Action != "" {
		u.Data = al["Data"]
		return nil
	}

	// RequiredLocation
	rl := map[string]RequiredLocation{}
	err = json.Unmarshal(data, &rl)
	if _, ok := err.(*json.UnmarshalTypeError); err != nil && !ok {
		return err
	}
	if rl["Data"].Number != 0 {
		u.Data = al["Data"]
		return nil
	}

	return nil
}

func (u Location) String() string {
	switch d := u.Data.(type) {
	case ActionLocation:
		return d.String()
	case RequiredLocation:
		return d.String()
	}
	return ""
}

func (r Result) String() string {
	var buffer strings.Builder
	buffer.WriteString(fmt.Sprintf("%s(%s) - %s\n", r.Level, r.Code, r.Title))
	buffer.WriteString(r.Location.String() + "\n")

	if r.Message != "" {
		buffer.WriteString(r.Message + "\n")
	}

	if r.URL != "" {
		buffer.WriteString(fmt.Sprintf("See %s for more information\n", r.URL))
	}

	buffer.WriteString("---\n")
	return buffer.String()
}

// ActionLocation identifies a location related to an action in a manifest.
type ActionLocation struct {
	// Action containing the step, e.g. Install.
	Action string

	// Mixin name, e.g. exec.
	Mixin string

	// StepNumber is the position of the step, starting from 1, within the action.
	// Example
	// install:
	//  - exec: (1)
	//     ...
	//  - helm: (2)
	//     ...
	//  - exec: (3)
	//     ...
	StepNumber int

	// StepDescription is the description of the step provided in the manifest.
	// Example
	// install:
	//  - exec:
	//      description: THIS IS THE STEP DESCRIPTION
	//      command: ./helper.sh
	StepDescription string
}

func (l ActionLocation) String() string {
	return fmt.Sprintf("%s: %s step in the %s mixin (%s)",
		l.Action, humanize.Ordinal(l.StepNumber), l.Mixin, l.StepDescription)
}

// Results is a set of items identified by the linter.
type Results []Result

func (r Results) String() string {
	var buffer strings.Builder
	// TODO: Sort, display errors first
	for _, result := range r {
		buffer.WriteString(result.String())
	}

	return buffer.String()
}

// HasError checks if any of the results is an error.
func (r Results) HasError() bool {
	for _, result := range r {
		if result.Level == LevelError {
			return true
		}
	}
	return false
}

// Linter manages executing the lint command for all affected mixins and reporting
// the results.
type Linter struct {
	*context.Context
	Mixins pkgmgmt.PackageManager
}

func New(cxt *context.Context, mixins pkgmgmt.PackageManager) *Linter {
	return &Linter{
		Context: cxt,
		Mixins:  mixins,
	}
}

func (l *Linter) Lint(m *manifest.Manifest) (Results, error) {
	var results Results

	if l.Debug {
		fmt.Fprintln(l.Err, "Linting the Porter manifest...")
	}
	requiredResults, err := l.lintRequired(m)
	if err != nil {
		return nil, errors.Wrap(err, "unable to lint the required section of the manifest")
	}
	results = append(results, requiredResults...)

	if l.Debug {
		fmt.Fprintln(l.Err, "Running linters for each mixin used in the manifest...")
	}

	q := query.New(l.Context, l.Mixins)
	responses, err := q.Execute("lint", query.NewManifestGenerator(m))
	if err != nil {
		return nil, err
	}

	for mixin, response := range responses {
		var r Results
		err = json.Unmarshal([]byte(response), &r)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to parse lint response from mixin %q", mixin)
		}

		results = append(results, r...)
	}

	return results, nil
}

// RequiredLocation identifies a location related to the required section in a manifest.
type RequiredLocation struct {
	// Number is the position of the required extension, starting from 1
	// Example
	// required:
	//  - docker: (1)
	//     ...
	//  - donuts: (2)
	//     ...
	Number int
}

func (l RequiredLocation) String() string {
	return fmt.Sprintf("%s extension in the required section",
		humanize.Ordinal(l.Number))
}

// CodeUnsupportedRequiredExtension is the linter code for an unsupported required extension
const CodeUnsupportedRequiredExtension = "required-100"

// TODO: should this live in pkg/cnab/extensions/required.go?  Or pkg/manifest?
func (l *Linter) lintRequired(m *manifest.Manifest) (Results, error) {
	results := make(Results, 0)

	for num, ext := range m.Required {
		supportedExt, err := extensions.GetSupportedExtension(ext.Name)
		if err != nil || supportedExt == nil {
			result := Result{
				Level: LevelWarning,
				Code:  CodeUnsupportedRequiredExtension,
				Location: Location{
					Data: RequiredLocation{
						Number: num + 1, // We index from 1 for natural counting, 1st, 2nd, etc.
					},
				},
				Title:   "Required Extensions: Unsupported Extension",
				Message: fmt.Sprintf("%q is not an extension currently supported by Porter", ext.Name),
				URL:     "https://porter.sh/author-bundles/#required",
			}
			results = append(results, result)
		}
	}

	return results, nil
}
