package provider

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
)

const DefaultMavenRepoUrl = "https://repo1.maven.org/maven2/"

const SnapshotVersionSuffix = "-SNAPSHOT"

type Repository struct {
	Url      string
	Username string
	Password string
}

type Artifact struct {
	GroupId    string
	ArtifactId string
	Version    string
	Classifier string
	Extension  string
}

type Metadata struct {
	Timestamp   string `xml:"versioning>snapshot>timestamp"`
	BuildNumber string `xml:"versioning>snapshot>buildNumber"`
}

func NewRepository(url, username, password string) *Repository {
	if url == "" {
		url = DefaultMavenRepoUrl
	}
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	return &Repository{
		Url:      url,
		Username: username,
		Password: password,
	}
}

func NewArtifact(groupId, artifactId, version, classifier, extension string) *Artifact {
	if extension == "" {
		extension = "jar"
	}
	return &Artifact{
		GroupId:    groupId,
		ArtifactId: artifactId,
		Version:    version,
		Classifier: classifier,
		Extension:  extension,
	}
}

func (a *Artifact) MetadataUrl(r *Repository) string {
	return r.Url + a.Path() + "maven-metadata.xml"
}

func (a *Artifact) Url(r *Repository, m *Metadata) string {
	return r.Url + a.Path() + a.FileName(m)
}

func (a *Artifact) ChecksumUrl(r *Repository, m *Metadata) string {
	return r.Url + a.Path() + a.FileName(m) + ".md5"
}

func (a *Artifact) Path() string {
	return fmt.Sprintf("%s/%s/%s/", strings.Replace(a.GroupId, ".", "/", -1), a.ArtifactId, a.Version)
}

func (a *Artifact) FileName(m *Metadata) string {
	version := a.Version
	if m != nil {
		version = version[0 : len(version)-len(SnapshotVersionSuffix)]
		version = fmt.Sprintf("%s-%s", version, m.SnapshotVersion())
	}
	if a.Classifier != "" {
		return fmt.Sprintf("%s-%s-%s.%s", a.ArtifactId, version, a.Classifier, a.Extension)
	} else {
		return fmt.Sprintf("%s-%s.%s", a.ArtifactId, version, a.Extension)
	}
}

func (r *Artifact) IsSnapshot() bool {
	return strings.HasSuffix(r.Version, SnapshotVersionSuffix)
}

func (r *Metadata) SnapshotVersion() string {
	return fmt.Sprintf("%s-%s", r.Timestamp, r.BuildNumber)
}

func DownloadMavenArtifact(repository *Repository, artifact *Artifact, outputPath string) (string, error) {
	if outputPath == "" {
		outputPath = artifact.FileName(nil)
	}

	var metadata *Metadata = nil
	if artifact.IsSnapshot() {
		metadataUrl := artifact.MetadataUrl(repository)
		resp, err := httpGet(metadataUrl, repository.Username, repository.Password)
		if err != nil {
			return "", err
		}
		if 400 <= resp.StatusCode {
			return "", errors.New(fmt.Sprintf("status code %d returned. URL: %s", resp.StatusCode, metadataUrl))
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		err = xml.Unmarshal(body, &metadata)
		if err != nil {
			return "", nil
		}
	}

	// 1. download checksum
	// 2. check file existance
	// 3. if not exists || checksum changed. donwload it
	checksum, err := downloadChecksum(repository, artifact, metadata)
	if err != nil {
		return "", err
	}

	checksumMatches := false
	fileExists := fileExists(outputPath)
	if fileExists {
		checksumMatches, err = verifyChecksum(outputPath, checksum)
		if err != nil {
			return "", err
		}
	}

	if fileExists && checksumMatches {
		return outputPath, nil
	}

	url := artifact.Url(repository, metadata)
	resp, err := httpGet(url, repository.Username, repository.Password)
	if err != nil {
		return "", err
	}
	if 400 <= resp.StatusCode {
		return "", errors.New(fmt.Sprintf("status code %d returned. URL: %s", resp.StatusCode, url))
	}
	defer resp.Body.Close()

	// default is current directory
	outputDir := path.Dir(outputPath)
	if outputDir == "" {
		outputDir = "."
	}
	// ensure outputDir is directory
	if _, err := os.Stat(outputDir); err != nil {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return "", err
		}
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return outputPath, nil
}

func downloadChecksum(repository *Repository, artifact *Artifact, metadata *Metadata) (string, error) {
	url := artifact.ChecksumUrl(repository, metadata)
	resp, err := httpGet(url, repository.Username, repository.Password)
	if err != nil {
		return "", err
	}
	if 400 <= resp.StatusCode {
		return "", errors.New(fmt.Sprintf("status code %d returned. URL: %s", resp.StatusCode, url))
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func httpGet(url, user, pwd string) (*http.Response, error) {
	if user != "" && pwd != "" {
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(user, pwd)
		return client.Do(req)
	}

	return http.Get(url)
}

func verifyChecksum(path string, expectedChecksum string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("Error while open file: %+w", err)
	}

	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	hasher := md5.New()

	if _, err := io.Copy(hasher, f); err != nil {
		return false, err
	}

	return hex.EncodeToString(hasher.Sum(nil)) == expectedChecksum, nil
}
