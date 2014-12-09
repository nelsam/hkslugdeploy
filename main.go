package main

import (
	"flag"
	"log"
	"os"

	"github.com/nelsam/hkslugdeploy/github"
	"github.com/nelsam/hkslugdeploy/heroku"
	"github.com/nelsam/hkslugdeploy/release"
)

const (
	herokuAppDir = "./app"
)

var (
	sequential        bool
	tarName           string
	githubReleaseName string
	githubReleaseDesc string
	githubRepo        string
	githubToken       string
	gitCommitish      string
	herokuApp         string
	herokuWebProc     string
	herokuEmail       string
	herokuKey         string
	selectedFilenames []string
)

func init() {
	flag.BoolVar(&sequential, "sequential", false,
		"Enable this flag if you don't want releases to run concurrently.")
	flag.StringVar(&tarName, "tarball-name", "release.tar.gz",
		"The name of the release tarball file that will be uploaded to github.")
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
	flag.StringVar(&herokuApp, "heroku-app", "",
		"Your heroku app's name.")
	flag.StringVar(&herokuWebProc, "heroku-web-proc", "",
		"The web process type to use when creating a heroku slug.")
	flag.StringVar(&herokuEmail, "heroku-email", "",
		"The email address for logging in to heroku.")
	flag.StringVar(&herokuKey, "heroku-password", "",
		"The password or access key to use when logging in to heroku.")
	flag.Parse()
	selectedFilenames = flag.Args()
}

func main() {
	release.CreateTarball(tarName, herokuAppDir, selectedFilenames)
	waiting := make([]chan bool, 0, 2)
	if githubRepo != "" {
		done := github.StartRelease(githubReleaseName, githubReleaseDesc, gitCommitish, githubRepo, githubToken, tarName)
		if sequential {
			<-done
		} else {
			waiting = append(waiting, done)
		}
	}
	if herokuApp != "" {
		done := heroku.StartRelease(herokuApp, map[string]string{"web": herokuWebProc}, herokuEmail, herokuKey, tarName)
		if sequential {
			<-done
		} else {
			waiting = append(waiting, done)
		}
	}
	for _, wait := range waiting {
		<-wait
	}
	if err := os.Remove(tarName); err != nil {
		log.Fatal(err)
	}
}
