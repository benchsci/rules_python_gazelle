package python

import (
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const (
	pyBinaryKind   = "py_binary"
	pyLibraryKind  = "py_library"
	pyTestKind     = "pytest"
	djangoTestKind = "django_test"
)

// Kinds returns a map that maps rule names (kinds) and information on how to
// match and merge attributes that may be found in rules of those kinds.
func (*Python) Kinds() map[string]rule.KindInfo {
	return pyKinds
}

var pyKinds = map[string]rule.KindInfo{
	pyBinaryKind: {
		MatchAny: false,
		NonEmptyAttrs: map[string]bool{
			"deps":       true,
			"main":       true,
			"srcs":       true,
			"imports":    true,
			"visibility": true,
		},
		MergeableAttrs: map[string]bool{
			"srcs": true,
		},
		ResolveAttrs: map[string]bool{
			"deps": true,
		},
	},
	pyLibraryKind: {
		MatchAny: false,
		NonEmptyAttrs: map[string]bool{
			"deps":       true,
			"srcs":       true,
			"imports":    true,
			"visibility": true,
		},
		MergeableAttrs: map[string]bool{
			"srcs": true,
		},
		ResolveAttrs: map[string]bool{
			"deps": true,
		},
	},
	pyTestKind: {
		MatchAny: false,
		NonEmptyAttrs: map[string]bool{
			"deps":       true,
			"main":       true,
			"srcs":       true,
			"imports":    true,
			"visibility": true,
		},
		MergeableAttrs: map[string]bool{
			"srcs": true,
		},
		ResolveAttrs: map[string]bool{
			"deps": true,
		},
	},
	djangoTestKind: {
		MatchAny: false,
		NonEmptyAttrs: map[string]bool{
			"deps":       true,
			"main":       true,
			"srcs":       true,
			"imports":    true,
			"visibility": true,
		},
		MergeableAttrs: map[string]bool{
			"srcs": true,
		},
		ResolveAttrs: map[string]bool{
			"deps": true,
		},
	},
}

// Loads returns .bzl files and symbols they define. Every rule generated by
// GenerateRules, now or in the past, should be loadable from one of these
// files.
func (py *Python) Loads() []rule.LoadInfo {
	return pyLoads
}

var pyLoads = []rule.LoadInfo{
	{
		Name: "@rules_python//python:defs.bzl",
		Symbols: []string{
			pyBinaryKind,
			pyLibraryKind,
			pyTestKind,
		},
	}, {
		Name: "@com_github_benchsci_rules_python_gazelle//:defs.bzl",
		Symbols: []string{
			djangoTestKind,
		},
	},
}
