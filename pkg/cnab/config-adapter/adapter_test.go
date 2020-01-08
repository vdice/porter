package configadapter

import (
	"fmt"
	"testing"

	"get.porter.sh/porter/pkg/cnab/extensions"
	"get.porter.sh/porter/pkg/config"
	"get.porter.sh/porter/pkg/manifest"
	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/bundle/definition"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestConverter_ToBundle(t *testing.T) {
	c := config.NewTestConfig(t)
	c.TestContext.AddTestFile("testdata/porter.yaml", config.Name)

	m, err := manifest.LoadManifestFrom(c.Context, config.Name)
	require.NoError(t, err, "could not load manifest")

	a := NewManifestConverter(c.Context, m, nil, nil)

	bun := a.ToBundle()

	assert.Equal(t, SchemaVersion, bun.SchemaVersion)
	assert.Equal(t, "hello", bun.Name)
	assert.Equal(t, "0.1.0", bun.Version)
	assert.Equal(t, "An example Porter configuration", bun.Description)

	stamp, err := LoadStamp(bun)
	assert.NoError(t, err, "could not load porter's stamp")
	assert.NotNil(t, stamp)

	assert.Contains(t, bun.Actions, "status", "custom action 'status' was not populated")
	assert.Contains(t, bun.Parameters, "porter-debug", "porter-debug parameter was not defined")
	assert.Contains(t, bun.Definitions, "porter-debug-parameter", "porter-debug definition was not defined")

	assert.Contains(t, bun.Custom, config.CustomBundleKey, "Porter stamp was not populated")
	assert.Contains(t, bun.Custom, extensions.DependenciesKey, "Dependencies was not populated")

	assert.Nil(t, bun.Outputs, "expected outputs section not to exist in generated bundle")
}

func TestManifestConverter_generateBundleParametersSchema(t *testing.T) {
	c := config.NewTestConfig(t)
	c.TestContext.AddTestFile("testdata/porter-with-parameters.yaml", config.Name)

	m, err := manifest.LoadManifestFrom(c.Context, config.Name)
	require.NoError(t, err, "could not load manifest")

	a := NewManifestConverter(c.Context, m, nil, nil)

	defs := make(definition.Definitions, len(m.Parameters))
	params := a.generateBundleParameters(&defs)

	testcases := []struct {
		propname  string
		wantParam bundle.Parameter
		wantDef   definition.Schema
	}{
		{"ainteger",
			bundle.Parameter{
				Definition: "ainteger-parameter",
				Destination: &bundle.Location{
					EnvironmentVariable: "AINTEGER",
				},
			},
			definition.Schema{
				Type:    "integer",
				Default: 1,
				Minimum: toInt(0),
				Maximum: toInt(10),
			},
		},
		{"anumber",
			bundle.Parameter{
				Definition: "anumber-parameter",
				Destination: &bundle.Location{
					EnvironmentVariable: "ANUMBER",
				},
			},
			definition.Schema{
				Type:             "number",
				Default:          0.5,
				ExclusiveMinimum: toInt(0),
				ExclusiveMaximum: toInt(1),
			},
		},
		{
			"astringenum",
			bundle.Parameter{
				Definition: "astringenum-parameter",
				Destination: &bundle.Location{
					EnvironmentVariable: "ASTRINGENUM",
				},
			},
			definition.Schema{
				Type:    "string",
				Default: "blue",
				Enum:    []interface{}{"blue", "red", "purple", "pink"},
			},
		},
		{
			"astring",
			bundle.Parameter{
				Definition: "astring-parameter",
				Destination: &bundle.Location{
					EnvironmentVariable: "ASTRING",
				},
				Required: true,
			},
			definition.Schema{
				Type:      "string",
				MinLength: toInt(1),
				MaxLength: toInt(10),
			},
		},
		{
			"aboolean",
			bundle.Parameter{
				Definition: "aboolean-parameter",
				Destination: &bundle.Location{
					EnvironmentVariable: "ABOOLEAN",
				},
			},
			definition.Schema{
				Type:    "boolean",
				Default: true,
			},
		},
		{
			"installonly",
			bundle.Parameter{
				Definition: "installonly-parameter",
				Destination: &bundle.Location{
					EnvironmentVariable: "INSTALLONLY",
				},
				ApplyTo: []string{
					"install",
				},
				Required: true,
			},
			definition.Schema{
				Type: "boolean",
			},
		},
		{
			"sensitive",
			bundle.Parameter{
				Definition: "sensitive-parameter",
				Destination: &bundle.Location{
					EnvironmentVariable: "SENSITIVE",
				},
				Required: true,
			},
			definition.Schema{
				Type:      "string",
				WriteOnly: toBool(true),
			},
		},
		{
			"jsonobject",
			bundle.Parameter{
				Definition: "jsonobject-parameter",
				Destination: &bundle.Location{
					EnvironmentVariable: "JSONOBJECT",
				},
			},
			definition.Schema{
				Type:    "string",
				Default: `"myobject": { "foo": "true", "bar": [ 1, 2, 3 ] }`,
			},
		},
		{
			"afile",
			bundle.Parameter{
				Definition: "afile-parameter",
				Destination: &bundle.Location{
					Path: "/root/.kube/config",
				},
				Required: true,
			},
			definition.Schema{
				Type:            "string",
				ContentEncoding: "base64",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.propname, func(t *testing.T) {
			param, ok := params[tc.propname]
			require.True(t, ok, "parameter definition was not generated")

			def, ok := defs[param.Definition]
			require.True(t, ok, "property definition was not generated")

			assert.Equal(t, tc.wantParam, param)
			assert.Equal(t, tc.wantDef, *def)
		})
	}
}

func TestManifestConverter_buildDefaultPorterParameters(t *testing.T) {
	c := config.NewTestConfig(t)
	c.TestContext.AddTestFile("../../manifest/testdata/simple.porter.yaml", config.Name)

	m, err := manifest.LoadManifestFrom(c.Context, config.Name)
	require.NoError(t, err, "could not load manifest")

	a := NewManifestConverter(c.Context, m, nil, nil)

	defs := make(definition.Definitions, len(m.Parameters))
	params := a.generateBundleParameters(&defs)

	debugParam, ok := params["porter-debug"]
	assert.True(t, ok, "porter-debug parameter was not defined")
	assert.Equal(t, "porter-debug-parameter", debugParam.Definition)
	assert.Equal(t, "PORTER_DEBUG", debugParam.Destination.EnvironmentVariable)

	debugDef, ok := defs["porter-debug-parameter"]
	require.True(t, ok, "porter-debug definition was not defined")
	assert.Equal(t, "boolean", debugDef.Type)
	assert.Equal(t, false, debugDef.Default)
}

func TestManifestConverter_generateImages(t *testing.T) {
	c := config.NewTestConfig(t)
	c.TestContext.AddTestFile("../../manifest/testdata/simple.porter.yaml", config.Name)

	m, err := manifest.LoadManifestFrom(c.Context, config.Name)
	require.NoError(t, err, "could not load manifest")

	a := NewManifestConverter(c.Context, m, nil, nil)

	mappedImage := manifest.MappedImage{
		Description: "un petite server",
		Repository:  "deislabs/myserver",
		ImageType:   "docker",
		Digest:      "abc123",
		Size:        12,
		MediaType:   "download",
		Labels: map[string]string{
			"OS":           "linux",
			"Architecture": "amd64",
		},
	}
	a.Manifest.ImageMap = map[string]manifest.MappedImage{
		"server": mappedImage,
	}

	images := a.generateBundleImages()

	require.Len(t, images, 1)
	img := images["server"]
	assert.Equal(t, mappedImage.Description, img.Description)
	assert.Equal(t, fmt.Sprintf("%s@%s", mappedImage.Repository, mappedImage.Digest), img.Image)
	assert.Equal(t, mappedImage.ImageType, img.ImageType)
	assert.Equal(t, mappedImage.Digest, img.Digest)
	assert.Equal(t, mappedImage.Size, img.Size)
	assert.Equal(t, mappedImage.MediaType, img.MediaType)
	assert.Equal(t, mappedImage.Labels["OS"], img.Labels["OS"])
	assert.Equal(t, mappedImage.Labels["Architecture"], img.Labels["Architecture"])
}

func TestManifestConverter_generateBundleImages_EmptyLabels(t *testing.T) {
	c := config.NewTestConfig(t)
	c.TestContext.AddTestFile("../../manifest/testdata/simple.porter.yaml", config.Name)

	m, err := manifest.LoadManifestFrom(c.Context, config.Name)
	require.NoError(t, err, "could not load manifest")

	a := NewManifestConverter(c.Context, m, nil, nil)

	mappedImage := manifest.MappedImage{
		Description: "un petite server",
		Repository:  "deislabs/myserver",
		Tag:         "1.0.0",
		ImageType:   "docker",
		Labels:      nil,
	}
	a.Manifest.ImageMap = map[string]manifest.MappedImage{
		"server": mappedImage,
	}

	images := a.generateBundleImages()
	require.Len(t, images, 1)
	img := images["server"]
	assert.Nil(t, img.Labels)
	assert.Equal(t, fmt.Sprintf("%s:%s", mappedImage.Repository, mappedImage.Tag), img.Image)
}

func TestManifestConverter_generateBundleOutputs(t *testing.T) {
	c := config.NewTestConfig(t)
	c.TestContext.AddTestFile("../../manifest/testdata/simple.porter.yaml", config.Name)

	m, err := manifest.LoadManifestFrom(c.Context, config.Name)
	require.NoError(t, err, "could not load manifest")

	a := NewManifestConverter(c.Context, m, nil, nil)

	outputDefinitions := []manifest.OutputDefinition{
		{
			Name: "output1",
			ApplyTo: []string{
				"install",
				"upgrade",
			},
			Schema: definition.Schema{
				Type:        "string",
				Description: "Description of output1",
			},
			Sensitive: true,
		},
		{
			Name: "output2",
			Schema: definition.Schema{
				Type:        "boolean",
				Description: "Description of output2",
			},
		},
		{
			Name: "kubeconfig",
			Path: "/root/.kube/config",
			Schema: definition.Schema{
				Type:        "file",
				Description: "Description of kubeconfig",
			},
		},
	}

	a.Manifest.Outputs = outputDefinitions

	defs := make(definition.Definitions, len(a.Manifest.Outputs))
	outputs := a.generateBundleOutputs(&defs)
	require.Len(t, defs, 3)

	wantOutputDefinitions := map[string]bundle.Output{
		"output1": {
			Definition:  "output1-output",
			Description: "Description of output1",
			ApplyTo: []string{
				"install",
				"upgrade",
			},
			Path: "/cnab/app/outputs/output1",
		},
		"output2": {
			Definition:  "output2-output",
			Description: "Description of output2",
			Path:        "/cnab/app/outputs/output2",
		},
		"kubeconfig": {
			Definition:  "kubeconfig-output",
			Description: "Description of kubeconfig",
			Path:        "/cnab/app/outputs/kubeconfig",
		},
	}

	require.Equal(t, wantOutputDefinitions, outputs)

	wantDefinitions := definition.Definitions{
		"output1-output": &definition.Schema{
			Type:        "string",
			Description: "Description of output1",
			WriteOnly:   toBool(true),
		},
		"output2-output": &definition.Schema{
			Type:        "boolean",
			Description: "Description of output2",
		},
		"kubeconfig-output": &definition.Schema{
			Type:            "string",
			ContentEncoding: "base64",
			Description:     "Description of kubeconfig",
		},
	}

	require.Equal(t, wantDefinitions, defs)
}

func TestManifestConverter_generateDependencies(t *testing.T) {
	c := config.NewTestConfig(t)
	c.TestContext.AddTestFile("testdata/porter-with-deps.yaml", config.Name)

	m, err := manifest.LoadManifestFrom(c.Context, config.Name)
	require.NoError(t, err, "could not load manifest")

	a := NewManifestConverter(c.Context, m, nil, nil)

	deps := a.generateDependencies()
	require.NotNil(t, deps, "Dependencies should not be nil")
	require.Len(t, deps.Requires, 3, "incorrect number of dependencies were generated")

	testcases := []struct {
		name    string
		wantDep extensions.Dependency
	}{
		{"no-version", extensions.Dependency{
			Bundle: "deislabs/azure-mysql:5.7",
		}},
		{"no-ranges", extensions.Dependency{
			Bundle: "deislabs/azure-active-directory",
			Version: &extensions.DependencyVersion{
				AllowPrereleases: true,
			},
		}},
		{"with-ranges", extensions.Dependency{
			Bundle: "deislabs/azure-blob-storage",
			Version: &extensions.DependencyVersion{
				Ranges: []string{
					"1.x - 2",
					"2.1 - 3.x",
				},
			},
		}},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var dep *extensions.Dependency
			for _, d := range deps.Requires {
				if d.Bundle == tc.wantDep.Bundle {
					dep = &d
					break
				}
			}

			require.NotNil(t, dep, "could not find bundle %s", tc.wantDep.Bundle)
			assert.Equal(t, &tc.wantDep, dep)
		})
	}
}

func TestManifestConverter_RequiredExtensions(t *testing.T) {
	c := config.NewTestConfig(t)
	c.TestContext.AddTestFile("testdata/porter-with-deps.yaml", config.Name)

	m, err := manifest.LoadManifestFrom(c.Context, config.Name)
	require.NoError(t, err, "could not load manifest")

	a := NewManifestConverter(c.Context, m, nil, nil)

	bun := a.ToBundle()

	assert.Equal(t, []string{"io.cnab.dependencies"}, bun.RequiredExtensions)
}

func TestManifestConverter_GenerateCustomActionDefinitions(t *testing.T) {
	c := config.NewTestConfig(t)
	c.TestContext.AddTestFile("testdata/porter-with-custom-action.yaml", config.Name)

	m, err := manifest.LoadManifestFrom(c.Context, config.Name)
	require.NoError(t, err, "could not load manifest")

	a := NewManifestConverter(c.Context, m, nil, nil)

	defs := a.generateCustomActionDefinitions()
	require.Len(t, defs, 2, "expected 2 custom action definitions to be generated")

	require.Contains(t, defs, "status")
	statusDef := defs["status"]
	assert.Equal(t, "Prints out status of world", statusDef.Description)
	assert.True(t, statusDef.Stateless, "expected the status custom action to be stateless")
	assert.False(t, statusDef.Modifies, "expected the status custom action to not modify resources")

	require.Contains(t, defs, "zombies")
	zombieDef := defs["zombies"]
	assert.Equal(t, "zombies", zombieDef.Description)
	assert.False(t, zombieDef.Stateless, "expected the zombies custom action to default to not stateless")
	assert.True(t, zombieDef.Modifies, "expected the zombies custom action to default to modifying resources")
}

func TestManifestConverter_generateDefaultAction(t *testing.T) {
	a := ManifestConverter{}

	testcases := []struct {
		action     string
		wantAction bundle.Action
	}{
		{"dry-run", bundle.Action{
			Description: "Execute the installation in a dry-run mode, allowing to see what would happen with the given set of parameter values",
			Modifies:    false,
			Stateless:   true,
		}},
		{
			"help", bundle.Action{
				Description: "Print an help message to the standard output",
				Modifies:    false,
				Stateless:   true,
			}},
		{"log", bundle.Action{
			Description: "Print logs of the installed system to the standard output",
			Modifies:    false,
			Stateless:   false,
		}},
		{"status", bundle.Action{
			Description: "Print a human readable status message to the standard output",
			Modifies:    false,
			Stateless:   false,
		}},
		{"zombies", bundle.Action{
			Description: "zombies",
			Modifies:    true,
			Stateless:   false,
		}},
	}

	for _, tc := range testcases {
		t.Run(tc.action, func(t *testing.T) {
			gotAction := a.generateDefaultAction(tc.action)
			assert.Equal(t, tc.wantAction, gotAction)
		})
	}
}
