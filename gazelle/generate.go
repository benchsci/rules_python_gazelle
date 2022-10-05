package python

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/emirpasic/gods/sets/treeset"
	godsutils "github.com/emirpasic/gods/utils"
	"github.com/google/uuid"

	"github.com/benchsci/rules_python_gazelle/gazelle/pythonconfig"
)

// GenerateRules extracts build metadata from source files in a directory.
// GenerateRules is called in each directory where an update is requested
// in depth-first post-order.
func (py *Python) GenerateRules(args language.GenerateArgs) language.GenerateResult {
	cfgs := args.Config.Exts[languageName].(pythonconfig.Configs)
	cfg := cfgs[args.Rel]

	if !cfg.ExtensionEnabled() {
		return language.GenerateResult{}
	}
	check_deps := treeset.NewWith(godsutils.StringComparator)

	pythonProjectRoot := cfg.PythonProjectRoot()

	pyFilenames := treeset.NewWith(godsutils.StringComparator)

	parser0 := newPython3Parser(args.Config.RepoRoot, args.Rel, cfg.IgnoresDependency)

	var result language.GenerateResult
	result.Gen = make([]*rule.Rule, 0)

	django_test_deps := treeset.NewWith(moduleComparator)
	django_test_files := treeset.NewWith(godsutils.StringComparator)

	for _, f := range args.RegularFiles {
		if cfg.IgnoresFile(filepath.Base(f)) {
			continue
		}
		ext := filepath.Ext(f)

		if ext != ".py" {
			continue
		}
		pyFilenames.Add(f)
	}
	parserOutput, err := parser0.parseMultipe(pyFilenames)
	if err != nil {
		log.Fatalf("ERROR: %v\n", err)
	}
	for _, parserOut := range parserOutput {
		f := parserOut.FileName
		ext := filepath.Ext(f)

		deps := parserOut.Modules
		targetName := strings.TrimSuffix(f, ext)

		if parserOut.RuleType == "py_test" {

			pyTestTarget := newTargetBuilder(getKind(args.Config, pyTestKind), targetName, pythonProjectRoot, args.Rel).
				addSrc(f).
				setMain(f).
				addModuleDependencies(deps)

			pyTest := pyTestTarget.build()
			check_deps.Add(label.Label{Repo: "", Pkg: "", Name: targetName, Relative: true}.String())

			result.Gen = append(result.Gen, pyTest)
			result.Imports = append(result.Imports, pyTest.PrivateAttr(config.GazelleImportsKey))
		} else if parserOut.RuleType == "py_binary" {

			pyBinaryTarget := newTargetBuilder(getKind(args.Config, pyBinaryKind), targetName, pythonProjectRoot, args.Rel).
				setMain(f).
				addSrc(f).
				addModuleDependencies(deps)

			pyBinary := pyBinaryTarget.build()

			result.Gen = append(result.Gen, pyBinary)
			result.Imports = append(result.Imports, pyBinary.PrivateAttr(config.GazelleImportsKey))
			check_deps.Add(label.Label{Repo: "", Pkg: "", Name: targetName, Relative: true}.String())
		} else if parserOut.RuleType == "django_test" {
			django_test_files.Add(f)
			it := deps.Iterator()
			for it.Next() {
				django_test_deps.Add(it.Value().(module))
			}

		} else {

			pyLibrary := newTargetBuilder(getKind(args.Config, pyLibraryKind), targetName, pythonProjectRoot, args.Rel).
				setUUID(uuid.Must(uuid.NewUUID()).String()).
				addSrc(f).
				addModuleDependencies(deps).
				build()

			result.Gen = append(result.Gen, pyLibrary)
			result.Imports = append(result.Imports, pyLibrary.PrivateAttr(config.GazelleImportsKey))
			check_deps.Add(label.Label{Repo: "", Pkg: "", Name: targetName, Relative: true}.String())
		}

	}
	if !django_test_files.Empty() {
		djangoTestTarget := newTargetBuilder(getKind(args.Config, djangoTestKind), "django_test", pythonProjectRoot, args.Rel).
			addSrcs(django_test_files).
			setConftest(cfg.PytestConfTest()).
			addModuleDependencies(django_test_deps).build()

		result.Gen = append(result.Gen, djangoTestTarget)
		result.Imports = append(result.Imports, djangoTestTarget.PrivateAttr(config.GazelleImportsKey))
		check_deps.Add(label.Label{Repo: "", Pkg: "", Name: "django_test", Relative: true}.String())
	}
	if !check_deps.Empty() && cfg.PyCheck() == "enabled" {
		pyCheck := newTargetBuilder(getKind(args.Config, pyCheckKind), "check", pythonProjectRoot, args.Rel).
			setUUID(uuid.Must(uuid.NewUUID()).String()).
			addSrcs(check_deps).
			build()

		result.Gen = append(result.Gen, pyCheck)
		result.Imports = append(result.Imports, pyCheck.PrivateAttr(config.GazelleImportsKey))
	}

	return result
}

func getKind(c *config.Config, kind_name string) string {
	// Extract kind_name from KindMap
	if kind, ok := c.KindMap[kind_name]; ok {
		return kind.KindName

	}
	return kind_name
}
