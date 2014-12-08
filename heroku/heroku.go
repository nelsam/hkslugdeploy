package heroku

import (
	"log"
	"net/http"
	"os"

	heroku "github.com/bgentry/heroku-go"
)

func Release(app string, procs map[string]string, email string, key string, build string) {
	client := heroku.Client{Username: email, Password: key}
	slug, err := client.SlugCreate(app, procs, nil)
	if err != nil {
		log.Fatal(err)
	}
	body, err := os.Open(build)
	if err != nil {
		log.Fatal(err)
	}
	defer body.Close()
	req, err := http.NewRequest(slug.Blob.Method, slug.Blob.URL, body)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-gtar")
	resp, err := new(http.Client).Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		log.Fatalf("Received status %s from slug upload", resp.Status)
	}
	_, err = client.ReleaseCreate(app, slug.Id, nil)
	if err != nil {
		log.Fatal(err)
	}
}
