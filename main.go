package main

import (
	"archive/tar"
	"crypto/subtle"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type tarStreamer struct {
	c    chan []byte
	done chan struct{}
}

func newTarStreamer(dir string) tarStreamer {
	return tarStreamer{
		c:    make(chan []byte),
		done: make(chan struct{}, 1),
	}
}

func (ts tarStreamer) Done() {
	ts.done <- struct{}{}
}

func (ts tarStreamer) Write(p []byte) (int, error) {
	ts.c <- p
	return len(p), nil
}

func (ts tarStreamer) Read(p []byte) (int, error) {
	select {
	case buf := <-ts.c:
		return copy(p, buf), nil
	case <-ts.done:
		return 0, io.EOF
	}
}

type writerDoner interface {
	Write(p []byte) (int, error)
	Done()
}

func buildTarArchive(baseDir string, wd writerDoner) {
	tw := tar.NewWriter(wd)

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, ferr error) error {

		var hdr *tar.Header

		//relativePath := strings.Replace(path, baseDir, "", 1)
		// the tar command does not include leading "/"
		// based on `tar cf - /tmp/dir | tar tvf -` output
		relativePath := path[1:]

		if info.IsDir() {
			hdr = &tar.Header{
				Name:     relativePath,
				Mode:     0700,
				ModTime:  info.ModTime(),
				Typeflag: tar.TypeDir,
			}
		} else {
			hdr = &tar.Header{
				Name:     relativePath,
				Mode:     0600,
				Size:     info.Size(),
				ModTime:  info.ModTime(),
				Typeflag: tar.TypeReg,
			}
		}

		if err := tw.WriteHeader(hdr); err != nil {
			log.Fatal(err)
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()

			_, err = io.Copy(tw, f)
			if err != nil {
				log.Fatal(err)
			}
		}

		if err := tw.Flush(); err != nil {
			log.Fatal(err)
		}

		return nil
	})
	if err != nil {
		panic(err)
	}

	tw.Close()
	wd.Done()
}

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

		// Tar pre
		tstreamer := newTarStreamer("useless")

		go buildTarArchive(baseDir, tstreamer)

		w.Header().Set("Content-Disposition", "attachment; filename=archive.tar")
		w.Header().Set("Content-Type", "application/x-tar")

		w.WriteHeader(http.StatusOK)

		io.Copy(w, tstreamer)
	}
}

func main() {
	http.HandleFunc("/", handlerFactory(os.Args[1], os.Args[2], os.Args[3]))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
