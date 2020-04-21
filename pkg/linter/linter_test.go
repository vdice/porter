package linter

import (
	"testing"

	"get.porter.sh/porter/pkg/context"
	"get.porter.sh/porter/pkg/manifest"
	"get.porter.sh/porter/pkg/mixin"
	"github.com/stretchr/testify/require"
)

func TestLinter_Lint(t *testing.T) {
	t.Run("no results", func(t *testing.T) {
		cxt := context.NewTestContext(t)
		mixins := mixin.NewTestMixinProvider()
		l := New(cxt.Context, mixins)
		m := &manifest.Manifest{
			Mixins: []manifest.MixinDeclaration{
				{
					Name: "exec",
				},
			},
		}
		mixins.LintResults = nil

		results, err := l.Lint(m)
		require.NoError(t, err, "Lint failed")
		require.Len(t, results, 0, "linter should have returned 0 results")
	})

	t.Run("has results", func(t *testing.T) {
		cxt := context.NewTestContext(t)
		mixins := mixin.NewTestMixinProvider()
		l := New(cxt.Context, mixins)
		m := &manifest.Manifest{
			Mixins: []manifest.MixinDeclaration{
				{
					Name: "exec",
				},
			},
		}
		mixins.LintResults = Results{
			{
				Level: LevelWarning,
				Code:  "exec-101",
				Title: "warning stuff isn't working",
			},
		}

		results, err := l.Lint(m)
		require.NoError(t, err, "Lint failed")
		require.Len(t, results, 1, "linter should have returned 1 result")
		require.Equal(t, mixins.LintResults, results, "unexpected lint results")
	})

	t.Run("mixin doesn't support lint", func(t *testing.T) {
		cxt := context.NewTestContext(t)
		mixins := mixin.NewTestMixinProvider()
		l := New(cxt.Context, mixins)
		m := &manifest.Manifest{
			Mixins: []manifest.MixinDeclaration{
				{
					Name: "nope",
				},
			},
		}

		results, err := l.Lint(m)
		require.NoError(t, err, "Lint failed")
		require.Len(t, results, 0, "linter should ignore mixins that doesn't support the lint command")
	})

}

func TestLinter_Lint_Required(t *testing.T) {
	t.Run("required extensions - supported", func(t *testing.T) {
		cxt := context.NewTestContext(t)
		mixins := mixin.NewTestMixinProvider()
		l := New(cxt.Context, mixins)
		m := &manifest.Manifest{
			Required: []manifest.RequiredExtension{
				{
					Name: "docker",
				},
			},
		}

		results, err := l.Lint(m)
		require.NoError(t, err, "Lint failed")
		require.Len(t, results, 0, "linter should not have returned any results")
	})

	t.Run("required extensions - unsupported", func(t *testing.T) {
		cxt := context.NewTestContext(t)
		mixins := mixin.NewTestMixinProvider()
		l := New(cxt.Context, mixins)
		m := &manifest.Manifest{
			Required: []manifest.RequiredExtension{
				{
					Name: "docker",
				},
				{
					Name: "foo",
				},
			},
		}
		lintResults := Results{
			{
				Level:   LevelWarning,
				Code:    CodeUnsupportedRequiredExtension,
				Title:   "Required Extensions: Unsupported Extension",
				Message: `"foo" is not an extension currently supported by Porter`,
				URL:     "https://porter.sh/author-bundles/#required",
				Location: Location{
					Data: RequiredLocation{
						Number: 2,
					},
				},
			},
		}

		results, err := l.Lint(m)
		require.NoError(t, err, "Lint failed")
		require.Len(t, results, 1, "linter should have returned 1 result")
		require.Equal(t, lintResults, results, "unexpected lint results")
		require.Contains(t, results[0].String(), "2nd extension in the required section")
	})
}
