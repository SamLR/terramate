package test

import (
	"testing"

	"github.com/madlambda/spells/assert"
	"github.com/mineiros-io/terrastack/git"
)

const (
	// Username for the test commits.
	Username = "terrastack tests"

	// Email for the test commits.
	Email = "terrastack@mineiros.io"
)

// NewGitWrapper tests the creation of a git wrapper and returns it if success.
func NewGitWrapper(t *testing.T, wd string, inheritEnv bool) *git.Git {
	t.Helper()

	gw, err := git.WithConfig(git.Config{
		Username:       Username,
		Email:          Email,
		WorkingDir:     wd,
		Isolated:       true,
		InheritEnv:     inheritEnv,
		AllowPorcelain: true,
	})
	assert.NoError(t, err, "new git wrapper")

	return gw
}

// EmptyRepo creates and initializes a git repository and checks for errors.
// If bare is provided, the repository is for revisions (ie: for pushs)
func EmptyRepo(t *testing.T, bare bool) string {
	t.Helper()

	gw := NewGitWrapper(t, "", false)

	repodir := t.TempDir()
	err := gw.Init(repodir, bare)
	assert.NoError(t, err, "git init")

	return repodir
}

// NewRepo creates and initializes a repository for terrastack use cases. It
// initializes two repositories, one for working and other bare for the
// "remote". It sets up the working repository with a "origin" remote pointing
// to the local "bare" repository and push a initial main commit onto
// origin/main. The working git repository is returned and the other is
// automatically cleaned up when the test function finishes.
func NewRepo(t *testing.T) string {
	t.Helper()

	repoDir := EmptyRepo(t, false)
	remoteDir := EmptyRepo(t, true)

	gw := NewGitWrapper(t, repoDir, false)

	err := gw.RemoteAdd("origin", remoteDir)
	assert.NoError(t, err, "git remote add origin")

	path := WriteFile(t, repoDir, "README.md", "# generated by terrastack tests")
	assert.NoError(t, gw.Add(path), "adding README.md to remote repo")
	assert.NoError(t, gw.Commit("add readme"))

	err = gw.Push("origin", "main")
	assert.NoError(t, err, "git push origin main")

	return repoDir
}