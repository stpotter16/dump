package handlers

import (
	"embed"
	"io/fs"
	"net/http"
	"strconv"
	"time"
)

//go:embed static
var staticFS embed.FS

func serveStaticFiles() http.HandlerFunc {

	fileSys, err := fs.Sub(staticFS, "static")

	if err != nil {
		panic(err)
	}

	server := http.FileServer(http.FS(fileSys))

	lastModifiedTime := time.Now()

	return func(w http.ResponseWriter, r *http.Request) {
		etag := "\"" + strconv.FormatInt(lastModifiedTime.UnixMilli(), 10) + "\""
		w.Header().Set("Etag", etag)
		w.Header().Set("Cache-Control", "max-age=3600")
		server.ServeHTTP(w, r)
	}
}
