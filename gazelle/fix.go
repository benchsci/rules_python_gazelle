package python

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Fix repairs deprecated usage of language-specific rules in f. This is
// called before the file is indexed. Unless c.ShouldFix is true, fixes
// that delete or rename rules should not be performed.
func (py *Python) Fix(c *config.Config, f *rule.File) {
	for _, r := range f.Rules {
		// delete deprecated js_import rule
		if r.Kind() == "old_django_test" {
			r.Delete()
		}
	}
	for _, l := range f.Loads {

		if l.Has("old_django_test") {
			l.Remove("old_django_test")
		}
	}
}
