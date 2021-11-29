package terrastack

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mineiros-io/terrastack/hcl"
	"github.com/mineiros-io/terrastack/hcl/hhcl"
)

// ConfigFilename is the name of the terrastack configuration file.
const ConfigFilename = "terrastack.tsk.hcl"

// Init initialize a stack. It's an error to initialize an already initialized
// stack unless they are of same versions. In case the stack is initialized with
// other terrastack version, the force flag can be used to explicitly initialize
// it anyway. The dir must be an absolute path.
func Init(dir string, force bool) error {
	if !filepath.IsAbs(dir) {
		// TODO(i4k): this needs to go away soon.
		return errors.New("init requires an absolute path")
	}
	st, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("init requires an existing directory")
		}

		return fmt.Errorf("stat failed on %q: %w", dir, err)
	}

	if !st.IsDir() {
		return errors.New("path is not a directory")
	}

	stackfile := filepath.Join(dir, ConfigFilename)
	isInitialized := false

	st, err = os.Stat(stackfile)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("stat failed on %q: %w", stackfile, err)
		}
	} else {
		isInitialized = true
	}

	if isInitialized && !st.Mode().IsRegular() {
		return fmt.Errorf("the path %q is not a regular file", stackfile)
	}

	if isInitialized && !force {
		version, err := parseVersion(stackfile)
		if err != nil {
			return fmt.Errorf("stack already initialized: error fetching "+
				"version: %w", err)
		}

		if version != Version() {
			return fmt.Errorf("stack already initialized with version %q "+
				"but terrastack version is %q", version, Version())
		}

		err = os.Remove(string(stackfile))
		if err != nil {
			return fmt.Errorf("while removing %q: %w", stackfile, err)
		}
	}

	f, err := os.Create(stackfile)
	if err != nil {
		return err
	}

	defer f.Close()

	var p hhcl.Printer
	err = p.PrintTerrastack(f, hcl.Terrastack{
		RequiredVersion: Version(),
	})

	if err != nil {
		return fmt.Errorf("failed to write %q: %w", stackfile, err)
	}

	return nil
}

func parseVersion(stackfile string) (string, error) {
	parser := hhcl.NewParser()
	ts, err := parser.ParseFile(stackfile)
	if err != nil {
		return "", fmt.Errorf("failed to parse file %q: %w", stackfile, err)
	}

	// TODO(i4k): properly support version constraints.
	ts.RequiredVersion = strings.TrimSpace(strings.TrimPrefix(ts.RequiredVersion, "~>"))
	return ts.RequiredVersion, nil
}