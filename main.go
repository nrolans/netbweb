package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nrolans/configstore"
	"github.com/nrolans/configstore/file"
)

func main() {

	// Store
	var store = (configstore.Store)(file.NewFileStore("/var/tmp/data", file.DefaultDateFormat))

	// HTTP endpoints
	mux := mux.NewRouter()
	mux.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/api", apiDoc)
	mux.Handle("/", http.RedirectHandler("/dashboard", 302))
	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, req *http.Request) {
		dashboard(store, w, req)
	})
	mux.HandleFunc("/hosts", func(w http.ResponseWriter, req *http.Request) {
		listHosts(store, w, req)
	})
	mux.HandleFunc("/hosts/{hostname}", func(w http.ResponseWriter, req *http.Request) {
		listDates(store, w, req)
	})
	mux.HandleFunc("/hosts/{hostname}/dates/{date}", func(w http.ResponseWriter, req *http.Request) {
		hostBackup(store, w, req)
	})
	mux.HandleFunc("/hosts/{hostname}/on/{date}", func(w http.ResponseWriter, req *http.Request) {
		showBackupDate(store, w, req)
	})
	mux.HandleFunc("/hosts/{hostname}/diff/{date1}/{date2}", func(w http.ResponseWriter, req *http.Request) {
		diffBackup(store, w, req)
	})

	// Start HTTP server
	s := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	s.ListenAndServe()

}
