package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

import (
	"os"
	"os/user"
)

const (
	herokuAppDir = "app"
	tarIsDir     = 040000
)

var (
	tarName           string
	herokuApp         string
	githubReleaseName string
	githubReleaseDesc string
	githubRepo        string
	githubToken       string
	gitCommitish      string
	herokuEmail       string
	herokuKey         string
	selectedFilenames []string
)

func init() {
	flag.StringVar(&tarName, "tarball-name", "release.tar.gz",
		"The name of the release tarball file that will be uploaded to github.")
	flag.StringVar(&herokuApp, "app", "",
		"Your heroku app's name.")
	flag.StringVar(&githubRepo, "github-repo", "",
		"Your github repo, in user/repo form.")
	flag.StringVar(&githubToken, "github-token", "",
		"Your github token for pushing a release.")
	flag.StringVar(&gitCommitish, "github-commitish", "",
		"The commitish that you're creating a github release of.")
	flag.StringVar(&githubReleaseName, "github-release-name", "",
		"The name to use when creating a release on github.")
	flag.StringVar(&githubReleaseDesc, "github-release-desc", "",
		"A description of this release for uploading to github.")
	flag.StringVar(&herokuEmail, "heroku-email", "",
		"The email address for logging in to heroku.")
	flag.StringVar(&herokuKey, "heroku-password", "",
		"The password or access key to use when logging in to heroku.")
	flag.Parse()
	selectedFilenames = flag.Args()
}

func main() {
	createTarball(tarName, selectedFilenames)
	githubRelease(githubReleaseName, githubReleaseDesc, gitCommitish, githubRepo, githubToken, tarName)
}

func createTarball(name string, srcFilenames []string) {
	srcFiles := make([]os.FileInfo, 0, len(selectedFilenames))
	for _, f := range srcFilenames {
		finfo, err := os.Stat(f)
		if err != nil {
			log.Fatal(err)
		}
		srcFiles = append(srcFiles, finfo)
	}
	targetFile, err := os.Create(name)
	if err != nil {
		log.Fatal(err)
	}
	gz := gzip.NewWriter(targetFile)
	tarball := tar.NewWriter(gz)
	defer func() {
		if err := tarball.Close(); err != nil {
			log.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			log.Fatal(err)
		}
		if err := targetFile.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	writeAppDir(tarball, herokuAppDir)
	archiveFiles(tarball, srcFiles, herokuAppDir)
}

func writeAppDir(tarball *tar.Writer, appDir string) {
	user, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	uid, err := strconv.Atoi(user.Uid)
	if err != nil {
		log.Fatal(err)
	}
	gid, err := strconv.Atoi(user.Gid)
	if err != nil {
		log.Fatal(err)
	}
	now := time.Now()
	header := &tar.Header{
		Name:       "app/",
		Mode:       int64(os.ModePerm | tarIsDir),
		Uid:        uid,
		Gid:        gid,
		ModTime:    now,
		AccessTime: now,
		ChangeTime: now,
		Typeflag:   tar.TypeDir,
	}
	tarball.WriteHeader(header)
}

func archiveFiles(tarball *tar.Writer, srcFiles []os.FileInfo, destPrefix string) {
	for _, finfo := range srcFiles {
		srcName := finfo.Name()
		dstName := path.Join(destPrefix, srcName)

		src, err := os.Open(srcName)
		if err != nil {
			log.Fatal(err)
		}
		defer src.Close()

		tarHeader, err := tar.FileInfoHeader(finfo, "")
		if err != nil {
			log.Fatal(err)
		}
		tarHeader.Name = path.Join(destPrefix, tarHeader.Name)
		tarball.WriteHeader(tarHeader)
		io.Copy(tarball, src)

		if finfo.IsDir() {
			contents, err := src.Readdir(0)
			if err != nil {
				log.Fatal(err)
			}
			archiveFiles(tarball, contents, dstName)
		}
	}
}

func githubRelease(releaseName string, releaseDesc string, commitish string, repo string, token string, attachments ...string) {
	uploadURL := githubReleaseUploadURL(releaseName, releaseDesc, commitish, repo, token)
	uploadURL = strings.Replace(uploadURL, "{?name}", "?name=%s", 1)
	for _, attachment := range attachments {
		githubUploadAttachment(uploadURL, attachment, token)
	}
}

func githubReleaseUploadURL(releaseName string, releaseDesc string, commitish string, repo string, token string) string {
	bodyMap := map[string]interface{}{
		"tag_name":         releaseName,
		"target_commitish": commitish,
		"name":             releaseName,
		"body":             releaseDesc,
		"draft":            true,
		"prerelease":       true,
	}
	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		log.Fatal(err)
	}
	body := bytes.NewBuffer(bodyBytes)
	resource := fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)
	req, err := http.NewRequest("POST", resource, body)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Content-Type", "application/json")
	resp, err := new(http.Client).Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		log.Fatalf("Github release returned status %s", resp.Status)
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var respFields map[string]interface{}
	if err := json.Unmarshal(respBytes, &respFields); err != nil {
		log.Fatal(err)
	}
	return respFields["upload_url"].(string)
}

func githubUploadAttachment(uploadURL string, attachment string, token string) {
	resource := fmt.Sprintf(uploadURL, attachment)
	body, err := os.Open(attachment)
	if err != nil {
		log.Fatal(err)
	}
	defer body.Close()
	req, err := http.NewRequest("POST", resource, body)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Content-Type", "application/x-gtar")
	resp, err := new(http.Client).Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		log.Fatalf("Github asset upload return status %s with response:\n%s", resp.Status, string(bodyBytes))
	}
}
