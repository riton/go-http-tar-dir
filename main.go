package main

import (
	"archive/tar"
	"crypto/subtle"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

		w.Header().Set("Content-Disposition", "attachment; filename=archive.tar")
		w.Header().Set("Content-Type", "application/x-tar")
		w.WriteHeader(http.StatusOK)

		tw := tar.NewWriter(w)
		err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			th, err := tar.FileInfoHeader(info, path)
			if err != nil {
				return err
			}
			fh, err := os.Open(path)
			if err != nil {
				return err
			}
			defer fh.Close()
			if err = tw.WriteHeader(th); err != nil {
				return err
			}
			_, err = io.Copy(tw, fh)
			return err
		})
		if err != nil {
			log.Fatal(err)
		}
	}
}

type options struct {
	url               string
	port              int
	host              string
	dirMode, fileMode string
	authCreds         string
	quitAfterN        int
}

func main() {

	var opt options

	flag.StringVar(&opt.url, "url", "/", "baseURL to serve")
	flag.StringVar(&opt.host, "listen", "", "adress to listen to (default to any)")
	flag.IntVar(&opt.port, "port", 8080, "port to listen to")
	flag.StringVar(&opt.dirMode, "dir-mode", "", "force directory mode")
	flag.StringVar(&opt.fileMode, "file-mode", "", "force file mode")
	flag.StringVar(&opt.authCreds, "basic-auth", "", "basic auth credentials in format `user:password`")
	flag.IntVar(&opt.quitAfterN, "quit-after", 0, "quit after serving N requests")
	flag.Parse()

	if len(flag.Args()) != 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	baseDir := flag.Arg(0)

	http.HandleFunc(opt.url, handlerFactory(baseDir))
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", opt.host, opt.port), nil))
}
