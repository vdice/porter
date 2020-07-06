package porter

import (
	"os"
	"path/filepath"
	"testing"

	"get.porter.sh/porter/pkg/build"
	"get.porter.sh/porter/pkg/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSharedOptions_defaultBundleFiles(t *testing.T) {
	cxt := context.NewTestContext(t)

	pwd, _ := os.Getwd()
	_, err := cxt.FileSystem.Create(filepath.Join(pwd, "porter.yaml"))
	require.NoError(t, err)

	opts := SharedOptions{}
	err = opts.defaultBundleFiles(cxt.Context)
	require.NoError(t, err)

	assert.Equal(t, "porter.yaml", opts.File)
	assert.Equal(t, build.LOCAL_BUNDLE, opts.CNABFile)
}

func TestSharedOptions_defaultBundleFiles_AltManifest(t *testing.T) {
	cxt := context.NewTestContext(t)

	opts := SharedOptions{
		bundleFileOptions: bundleFileOptions{
			File: "mybun/porter.yaml",
		},
	}
	err := opts.defaultBundleFiles(cxt.Context)
	require.NoError(t, err)

	assert.Equal(t, filepath.Join("mybun", build.LOCAL_BUNDLE), opts.CNABFile)
}

func TestSharedOptions_validateBundleJson(t *testing.T) {
	pwd, _ := os.Getwd()

	cxt := context.NewTestContext(t)

	cxt.FileSystem.Create("/mybun1/bundle.json")
	cxt.FileSystem.Create(filepath.Join(pwd, "bundle1.json"))

	testcases := []struct {
		name           string
		cnabFile       string
		wantBundleJson string
		wantError      string
	}{
		{name: "absolute file exists", cnabFile: "/mybun1/bundle.json", wantBundleJson: "/mybun1/bundle.json", wantError: ""},
		{name: "relative file exists", cnabFile: "bundle1.json", wantBundleJson: filepath.Join(pwd, "bundle1.json"), wantError: ""},
		{name: "absolute file does not exist", cnabFile: "/mybun2/bundle.json", wantError: "unable to access --cnab-file /mybun2/bundle.json"},
		{name: "relative file does not", cnabFile: "bundle2.json", wantError: "unable to access --cnab-file bundle2.json"},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			opts := SharedOptions{
				bundleFileOptions: bundleFileOptions{
					CNABFile: tc.cnabFile,
				},
			}

			err := opts.validateCNABFile(cxt.Context)

			if tc.wantError == "" {
				require.NoError(t, err)
				assert.Equal(t, opts.CNABFile, tc.wantBundleJson)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantError)
			}
		})
	}
}

func TestSharedOptions_defaultDriver(t *testing.T) {
	opts := SharedOptions{}

	opts.defaultDriver()

	assert.Equal(t, DefaultDriver, opts.Driver)
}
