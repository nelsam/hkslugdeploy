package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/nelsam/hkslugdeploy/curl"
)

func Release(releaseName string, releaseDesc string, commitish string, repo string, token string, attachments ...string) {
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
		log.Printf("Github asset upload return status %s with response:\n%s", resp.Status, string(bodyBytes))

		// This is frustrating ... I can't figure out why, but go's requests
		// usually fail when performing the asset upload, while curl works
		// fine when sending what I believe to be the exact same request.
		log.Print("Trying again with shell call to curl...")
		cmd := curl.Command(req)
		if output, err := cmd.Output(); err != nil {
			log.Fatalf("Received error %s from curl, with output:\n%s", err, string(output))
		}
	}
}
