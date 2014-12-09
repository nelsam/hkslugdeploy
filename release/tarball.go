package release

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"log"
	"os"
	"os/user"
	"path"
	"strconv"
	"time"
)

const (
	tarIsDir = 040000
)

func CreateTarball(name string, topLevelFolder string, srcFilenames []string) {
	log.Print("[release] Creating tarball")
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

	log.Print("[release] Writing fake top level directory to tarball")
	writeAppDir(tarball, topLevelFolder)

	log.Print("[release] Adding requested files to archive")
	for _, f := range srcFilenames {
		finfo, err := os.Stat(f)
		if err != nil {
			log.Fatal(err)
		}
		dir := path.Dir(f)
		archiveFiles(tarball, []os.FileInfo{finfo}, dir, topLevelFolder)
	}
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
		// "./" needs to preceed all files in a heroku slug.
		Name:       "./" + appDir,
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

func archiveFiles(tarball *tar.Writer, srcFiles []os.FileInfo, directory string, destPrefix string) {
	for _, finfo := range srcFiles {
		srcName := finfo.Name()
		if directory != "" {
			srcName = path.Join(directory, srcName)
		}

		src, err := os.Open(srcName)
		if err != nil {
			log.Fatal(err)
		}
		defer src.Close()

		tarHeader, err := tar.FileInfoHeader(finfo, "")
		if err != nil {
			log.Fatal(err)
		}
		// As in writeAppDir, "./" always needs to preceed entries in a
		// heroku slug.
		tarHeader.Name = "./" + path.Join(destPrefix, srcName)
		tarball.WriteHeader(tarHeader)
		io.Copy(tarball, src)

		if finfo.IsDir() {
			contents, err := src.Readdir(0)
			if err != nil {
				log.Fatal(err)
			}
			archiveFiles(tarball, contents, srcName, destPrefix)
		}
	}
}
