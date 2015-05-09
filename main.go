package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nelsam/hkslugdeploy/github"
	"github.com/nelsam/hkslugdeploy/heroku"
	"github.com/nelsam/hkslugdeploy/release"
)

const (
	herokuAppDir = "app"
)

var (
	sequential        bool
	clean             bool
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
	herokuWorkerProc  string
	selectedFilenames []string
)

func init() {
	flag.BoolVar(&sequential, "sequential", false,
		"Enable this flag if you don't want releases to run concurrently.")
	flag.BoolVar(&clean, "clean", false,
		"Clean up any files that are created during deployment (including the release tarball).")
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
	flag.StringVar(&herokuEmail, "heroku-login", "",
		"The email address for logging in to heroku.")
	flag.StringVar(&herokuKey, "heroku-token", "",
		"The password or access token to use when logging in to heroku.")
	flag.StringVar(&herokuWorkerProc, "heroku-worker-proc", "",
		"The name of the worker process to execute. Must be provided in the list of files to include in the binary.")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [options] release_files...\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Creates a heroku slug tarball (.tar.gz) out of the provided release_files, and "+
			"optionally uploads the slug tarball to a github release and/or deploys it to heroku.  Make sure "+
			"that you provide *all* github options if you want to create a github release with the resulting "+
			"tarball, and the same for heroku if you want to deploy the slug to heroku.\n\n")
		fmt.Fprint(os.Stderr, "options:\n")
		flag.PrintDefaults()
	}
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
		procs := map[string]string{"web": herokuWebProc}
		if herokuWorkerProc != "" {
			procs["worker"] = herokuWorkerProc
		}
		done := heroku.StartRelease(herokuApp, procs, herokuEmail, herokuKey, tarName, gitCommitish)
		if sequential {
			<-done
		} else {
			waiting = append(waiting, done)
		}
	}
	for _, wait := range waiting {
		<-wait
	}
	if clean {
		if err := os.Remove(tarName); err != nil {
			log.Fatal(err)
		}
	}
}
