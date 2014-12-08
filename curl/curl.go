package curl

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func Command(req *http.Request) *exec.Cmd {
	// I chose a buffer of len(req.Header)*2 since many header entries
	// can have multiple values, so this gets us closer to having enough
	// room for everything, without going overboard.  Hopefully.
	args := make([]string, 0, len(req.Header)*2)
	args = append(args, "-X", req.Method)
	for name, values := range req.Header {
		for _, value := range values {
			args = append(args, "-H", fmt.Sprintf("%s: %s", name, value))
		}
	}
	if req.Body != nil {
		// Body must be a file for us to be able to use it.
		filename := req.Body.(*os.File).Name()
		args = append(args, "--data-binary", fmt.Sprintf("@%s", filename))
	}
	args = append(args, req.URL.String())
	log.Printf("Created curl command: %s", fmt.Sprintf("curl %s", strings.Join(args, " ")))
	return exec.Command("curl", args...)
}
