package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"io"
	"log"
	"path"
	"strconv"
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
	releaseName       string
	herokuApp         string
	githubRepo        string
	githubToken       string
	herokuEmail       string
	herokuKey         string
	selectedFilenames []string
)

func init() {
	flag.StringVar(&releaseName, "tarball-name", "release.tar.gz",
		"The name of the release tarball file that will be uploaded "+
			"to github.")
	flag.StringVar(&herokuApp, "app", "",
		"Your heroku app's name.")
	flag.StringVar(&githubRepo, "github-repo", "",
		"Your github repo, in user/repo form.")
	flag.StringVar(&githubToken, "github-token", "",
		"Your github token for pushing a release.")
	flag.StringVar(&herokuEmail, "heroku-email", "",
		"The email address for logging in to heroku.")
	flag.StringVar(&herokuKey, "heroku-password", "",
		"The password or access key to use when logging in to heroku.")
	flag.Parse()
	selectedFilenames = flag.Args()
}

func main() {
	createTarball(releaseName, selectedFilenames)
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
	defer targetFile.Close()
	gz := gzip.NewWriter(targetFile)
	defer gz.Close()
	tarball := tar.NewWriter(gz)
	defer tarball.Close()
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
	log.Printf("%v", header)
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
