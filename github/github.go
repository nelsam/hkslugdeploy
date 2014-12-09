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

func StartRelease(releaseName string, releaseDesc string, commitish string, repo string, token string, assets ...string) chan bool {
	done := make(chan bool)
	go func() {
		Release(releaseName, releaseDesc, commitish, repo, token, assets...)
		done <- true
	}()
	return done
}

func Release(releaseName string, releaseDesc string, commitish string, repo string, token string, assets ...string) {
	log.Print("[github] Creating github release")
	uploadURL := releaseUploadURL(releaseName, releaseDesc, commitish, repo, token)
	uploadURL = strings.Replace(uploadURL, "{?name}", "?name=%s", 1)
	for _, asset := range assets {
		log.Printf("[github] Uploading asset %s", asset)
		uploadAsset(uploadURL, asset, token)
	}
	log.Print("[github] Done")
}

func releaseUploadURL(releaseName string, releaseDesc string, commitish string, repo string, token string) string {
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
		log.Fatalf("[github][error] %s", err)
	}
	body := bytes.NewBuffer(bodyBytes)
	resource := fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)
	req, err := http.NewRequest("POST", resource, body)
	if err != nil {
		log.Fatalf("[github][error] %s", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Content-Type", "application/json")
	resp, err := new(http.Client).Do(req)
	if err != nil {
		log.Fatalf("[github][error] %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		log.Fatalf("[github][error] Release returned status %s", resp.Status)
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("[github][error] %s", err)
	}
	var respFields map[string]interface{}
	if err := json.Unmarshal(respBytes, &respFields); err != nil {
		log.Fatalf("[github][error] %s", err)
	}
	return respFields["upload_url"].(string)
}

func uploadAsset(uploadURL string, asset string, token string) {
	resource := fmt.Sprintf(uploadURL, asset)
	body, err := os.Open(asset)
	if err != nil {
		log.Fatalf("[github][error] %s", err)
	}
	defer body.Close()
	req, err := http.NewRequest("POST", resource, body)
	if err != nil {
		log.Fatalf("[github][error] %s", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Content-Type", "application/x-gtar")
	resp, err := new(http.Client).Do(req)
	if err != nil {
		log.Fatalf("[github][error] %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		log.Printf("[github][warning] Github asset upload returned status %s with response:\n%s", resp.Status, string(bodyBytes))

		// This is frustrating ... I can't figure out why, but go's requests
		// usually fail when performing the asset upload, while curl works
		// fine when sending what I believe to be the exact same request.
		log.Print("[github] Trying again with shell call to curl...")
		cmd := curl.Command(req)
		if output, err := cmd.Output(); err != nil {
			log.Fatalf("[github][error] Received error %s from curl, with output:\n%s", err, string(output))
		}
	}
}
