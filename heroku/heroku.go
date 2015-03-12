package heroku

import (
	"log"
	"net/http"
	"os"
	"strings"

	heroku "github.com/bgentry/heroku-go"
	"github.com/nelsam/hkslugdeploy/curl"
)

func StartRelease(app string, procs map[string]string, email string, key string, build string, commitish string) chan bool {
	done := make(chan bool)
	go func() {
		Release(app, procs, email, key, build, commitish)
		done <- true
	}()
	return done
}

func Release(app string, procs map[string]string, email string, key string, build string, commitish string) {
	client := &heroku.Client{Username: email, Password: key}
	log.Print("[heroku] Creating release slug")
	slug, err := client.SlugCreate(app, procs, &heroku.SlugCreateOpts{Commit: &commitish})
	if err != nil {
		log.Fatalf("[heroku][error] %s", err)
	}
	body, err := os.Open(build)
	if err != nil {
		log.Fatalf("[heroku][error] %s", err)
	}
	defer body.Close()
	slug.Blob.Method = strings.ToUpper(slug.Blob.Method)
	req, err := http.NewRequest(slug.Blob.Method, slug.Blob.URL, body)
	if err != nil {
		log.Fatalf("[heroku][error] %s", err)
	}

	// Because Amazon has one of the worst APIs on the planet, we can't
	// send a proper request to it.  It requires the Content-Type header
	// to be empty, but Go appears to correct empty Content-Type headers
	// (although I can't find out exactly where this is happening).
	// Instead, we need to send a request via curl.
	cmd := curl.Command(req, "-H", "Content-Type:")
	log.Print("[heroku] Uploading build")
	output, err := cmd.Output()
	if err != nil || strings.Contains(string(output), "<error>") {
		log.Fatalf("[heroku][error] Received curl error %v with body:\n%s", err, string(output))
	}
	log.Printf("[heroku] Publishing release")
	if _, err = client.ReleaseCreate(app, slug.Id, nil); err != nil {
		log.Fatalf("[heroku][error] %s", err)
	}
	log.Print("[heroku] Done")
}
