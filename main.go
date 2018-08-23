package main

import (
	"crypto/subtle"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
)

func handlerFactory(baseDir string, username string, password string) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		user, pass, ok := r.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
			w.Header().Set("WWW-Authenticate", "Basic realm=restricted")
			w.WriteHeader(401)
			return
		}

		tar_cmd := exec.Command("tar", "cf", "-", baseDir)

		tar, err := tar_cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
		defer tar.Close()

		if err := tar_cmd.Start(); err != nil {
			panic(err)
		}
		defer tar_cmd.Wait()

		w.Header().Set("Content-Disposition", "attachment; filename=archive.tar")
		w.Header().Set("Content-Type", "application/x-tar")

		w.WriteHeader(http.StatusOK)

		io.Copy(w, tar)
	}
}

func main() {
	http.HandleFunc("/", handlerFactory(os.Args[1], os.Args[2], os.Args[3]))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
