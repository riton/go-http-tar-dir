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
	"strings"
)

type options struct {
	url                    string
	port                   int
	host                   string
	dirMode, fileMode      int64
	authCreds              string
	quitAfterFirstDownload bool
	rewriteBaseDir         string
	excludeExt             stringArray
}

func authHandle(w http.ResponseWriter, r *http.Request, opt *options) bool {

	creds := strings.SplitN(opt.authCreds, ":", 2)
	username, password := creds[0], creds[1]

	user, pass, ok := r.BasicAuth()
	if !ok ||
		subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 ||
		subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
		w.Header().Set("WWW-Authenticate", "Basic realm=restricted")
		w.WriteHeader(401)
		return false
	}

	return true
}

func excludeFileByExtension(path string, excludedExt []string) bool {

	pathExt := filepath.Ext(path)

	for _, ext := range excludedExt {
		if pathExt == ext {
			fmt.Printf("file '%s' excluded by file extension\n", path)
			return true
		}
	}

	return false
}

func handlerFactory(baseDir string, opt *options) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		fmt.Printf("Serving request for %s", r.RemoteAddr)

		if opt.authCreds != "" {
			if !authHandle(w, r, opt) {
				return
			}
		}

		w.Header().Set("Content-Disposition", "attachment; filename=archive.tar")
		w.Header().Set("Content-Type", "application/x-tar")
		w.WriteHeader(http.StatusOK)

		tw := tar.NewWriter(w)
		err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if excludeFileByExtension(path, opt.excludeExt) {
				return nil
			}

			th, err := tar.FileInfoHeader(info, path)
			if err != nil {
				return err
			}

			// Restore full file path
			if opt.rewriteBaseDir == "" {
				th.Name = path
			} else {
				th.Name = strings.Replace(path, baseDir, opt.rewriteBaseDir, 1)
			}

			// Enforce file or dir mode if requested
			if info.IsDir() {
				if opt.dirMode > 0 {
					th.Mode = opt.dirMode
				}
			} else {
				if opt.fileMode > 0 {
					th.Mode = opt.fileMode
				}
			}

			if err = tw.WriteHeader(th); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			fh, err := os.Open(path)
			if err != nil {
				return err
			}
			defer fh.Close()

			_, err = io.Copy(tw, fh)
			return err
		})
		if err != nil {
			log.Fatal(err)
		}

		if err := tw.Close(); err != nil {
			log.Fatal(err)
		}

		if opt.quitAfterFirstDownload {
			os.Exit(0)
		}
	}
}

type stringArray []string

func (i *stringArray) String() string {
	return strings.Join(*i, ",")
}
func (i *stringArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {

	var opt options

	flag.StringVar(&opt.url, "url", "/", "baseURL to serve")
	flag.StringVar(&opt.host, "listen", "", "adress to listen to (default to any)")
	flag.IntVar(&opt.port, "port", 8080, "port to listen to")
	flag.Int64Var(&opt.dirMode, "dir-mode", -1, "force directory mode")
	flag.Int64Var(&opt.fileMode, "file-mode", -1, "force file mode")
	flag.StringVar(&opt.authCreds, "basic-auth", "", "basic auth credentials in format `user:password`")
	flag.StringVar(&opt.rewriteBaseDir, "rewrite-base-dir", "", "rewrite base dir to something else. Do not expose local filepath.")
	flag.BoolVar(&opt.quitAfterFirstDownload, "quit-after", false, "quit after first download")
	flag.Var(&opt.excludeExt, "exclude-extension", "Extension to exclude from archive")
	flag.Parse()

	if len(flag.Args()) != 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	baseDir := flag.Arg(0)

	http.HandleFunc(opt.url, handlerFactory(baseDir, &opt))
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", opt.host, opt.port), nil))
}
