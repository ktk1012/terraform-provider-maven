package provider

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestDownloadFromMavenCentral(t *testing.T) {
	td, cwd := setup(t)
	defer tearDown(t, td, cwd)

	r := NewRepository("", "", "")
	a := NewArtifact("org.apache.commons", "commons-text", "1.9", "", "")
	path, err := DownloadMavenArtifact(r, a, "")
	assert.Equal(t, "commons-text-1.9.jar", path)
	assert.Nil(t, err)
	fi, err := os.Stat(path)
	assert.Positive(t, fi.Size())
}

func TestDownloadWithOutputPath(t *testing.T) {
	td, cwd := setup(t)
	defer tearDown(t, td, cwd)

	r := NewRepository("", "", "")
	a := NewArtifact("org.apache.commons", "commons-text", "1.9", "javadoc", "")
	path, err := DownloadMavenArtifact(r, a, "out/path.jar")
	assert.Equal(t, "out/path.jar", path)
	assert.Nil(t, err)
	fi, err := os.Stat(path)
	assert.Positive(t, fi.Size())
}

func TestDownloadNotFound(t *testing.T) {
	td, cwd := setup(t)
	defer tearDown(t, td, cwd)

	r := NewRepository("", "", "")
	a := NewArtifact("invalid.group", "commons-text", "1.9", "", "")
	path, err := DownloadMavenArtifact(r, a, "")
	assert.Equal(t, "", path)
	assert.ErrorContains(t, err, "status code 404 returned. URL: https://repo1.maven.org/maven2/invalid/group/commons-text/1.9/commons-text-1.9")
}

func TestDownloadFromPrivateRepository(t *testing.T) {
	td, cwd := setup(t)
	defer tearDown(t, td, cwd)

	r := NewRepository("", "", "")
	a := NewArtifact("invalid.group", "commons-text", "1.9", "", "")
	path, err := DownloadMavenArtifact(r, a, "")
	assert.Equal(t, "", path)
	assert.ErrorContains(t, err, "status code 404 returned. URL: https://repo1.maven.org/maven2/invalid/group/commons-text/1.9/commons-text-1.9")
}

func TestDownloadSnapshot(t *testing.T) {
	td, cwd := setup(t)
	defer tearDown(t, td, cwd)

	r := NewRepository("https://repository.apache.org/content/repositories/snapshots", "", "")
	a := NewArtifact("org.apache.commons", "commons-text", "1.10.1-SNAPSHOT", "", "")
	path, err := DownloadMavenArtifact(r, a, "")
	assert.Equal(t, "commons-text-1.10.1-SNAPSHOT.jar", path)
	assert.Nil(t, err)
	fi, err := os.Stat(path)
	assert.Positive(t, fi.Size())
}

func TestDownloadTwice(t *testing.T) {
	td, cwd := setup(t)
	defer tearDown(t, td, cwd)

	r := NewRepository("", "", "")
	a := NewArtifact("org.apache.commons", "commons-text", "1.9", "", "")
	path, err := DownloadMavenArtifact(r, a, "")
	assert.Equal(t, "commons-text-1.9.jar", path)
	assert.Nil(t, err)
	fi1, err := os.Stat(path)
	assert.Positive(t, fi1.Size())

	path, err = DownloadMavenArtifact(r, a, "")
	assert.Equal(t, "commons-text-1.9.jar", path)
	assert.Nil(t, err)
	fi2, err := os.Stat(path)
	assert.Equal(t, fi1.ModTime(), fi2.ModTime())
}
