package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/hoisie/mustache"
	"github.com/nrolans/configstore"
	"github.com/nrolans/configstore/file"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func dashboard(store configstore.Store, w http.ResponseWriter, req *http.Request) {

	names, err := store.Names()
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, "error :(", 500)
		return
	}

	if req.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-type", "application/json")

		var data = []interface{}{}

		for _, name := range names {
			dates, err := store.Dates(name)
			if err != nil {
				continue
			}

			var date *time.Time
			if len(dates) > 0 {
				date = &dates[0]
			} else {
				date = nil
			}

			data = append(data, struct {
				Hostname   string
				LastBackup *time.Time `json:",omitempty"`
			}{
				Hostname:   name,
				LastBackup: date,
			})

		}

		enc := json.NewEncoder(w)
		err = enc.Encode(data)

	} else {

		var data = struct {
			Entries []struct {
				Hostname   string
				LastBackup *time.Time
				Ago        string
				AgoStatus  string
			}
		}{}

		for _, name := range names {
			dates, err := store.Dates(name)
			if err != nil {
				continue
			}

			var date *time.Time
			if len(dates) > 0 {
				date = &dates[0]
			} else {
				date = nil
			}

			data.Entries = append(data.Entries, struct {
				Hostname   string
				LastBackup *time.Time
				Ago        string
				AgoStatus  string
			}{
				Hostname:   name,
				LastBackup: date,
				Ago:        dashboardAgo(date, 24, 72),
				AgoStatus:  dashboardAgoStatus(date, 24, 72),
			})
		}

		io.WriteString(w, mustache.RenderFileInLayout("templates/dashboard.html.mustache", "templates/layout.html.mustache", data))
	}
}

func dashboardAgo(t *time.Time, warning, danger int) string {
	if t == nil {
		return "No backup"
	}
	d := time.Since(*t)
	if d.Hours() > float64(danger) {
		return fmt.Sprintf("%d days ago", int(d.Hours())/24)
	} else if d.Hours() > float64(warning) {
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	}
	return fmt.Sprintf("%d hours ago", int(d.Hours()))
}

func dashboardAgoStatus(t *time.Time, warning, danger int) string {
	if t == nil {
		return "default"
	}
	d := time.Since(*t)
	if d.Hours() > float64(danger) {
		return "danger"
	} else if d.Hours() > float64(warning) {
		return "warning"
	}
	return "success"
}

func apiDoc(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, mustache.RenderFileInLayout("templates/api.html.mustache", "templates/layout.html.mustache", nil))
}

func listHosts(store configstore.Store, w http.ResponseWriter, req *http.Request) {
	names, err := store.Names()
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, "error :(", 500)
		return
	}

	if req.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-type", "application/json")

		enc := json.NewEncoder(w)
		err := enc.Encode(struct {
			Hostnames []string
		}{
			Hostnames: names,
		})

		if err != nil {
			log.Printf("Error: %s", err)
			http.Error(w, "error :(", 500)
			return
		}
	} else {
		io.WriteString(w, mustache.RenderFileInLayout("templates/hosts.html.mustache", "templates/layout.html.mustache", names))
	}
}

func listDates(store configstore.Store, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	datesStore, err := store.Dates(vars["hostname"])
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, "error :(", 500)
		return
	}

	if req.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-type", "application/json")

		var dates []string
		for _, date := range datesStore {
			dates = append(dates, date.Format(file.DefaultDateFormat))
		}

		enc := json.NewEncoder(w)
		err := enc.Encode(struct {
			Hostname string
			Dates    []string
		}{
			Hostname: vars["hostname"],
			Dates:    dates,
		})
		if err != nil {
			log.Printf("Error: %s", err)
			http.Error(w, "error :(", 500)
			return
		}
	} else {

		var data = struct {
			Hostname string
			Dates    []struct {
				String string
				URL    string
			}
		}{
			Hostname: vars["hostname"],
		}
		for _, date := range datesStore {
			data.Dates = append(data.Dates, struct {
				String string
				URL    string
			}{
				String: date.String(),
				URL:    date.Format(file.DefaultDateFormat),
			})
		}

		io.WriteString(w, mustache.RenderFileInLayout("templates/dates.html.mustache", "templates/layout.html.mustache", data))
	}
}

func hostBackup(store configstore.Store, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	t, err := time.Parse(file.DefaultDateFormat, vars["date"])
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, "error :(", 500)
		return
	}

	e := configstore.Entry{
		Name: vars["hostname"],
		Date: t,
	}

	err = store.Get(&e)
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, "error :(", 500)
		return
	}

	if req.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-type", "application/json")

		enc := json.NewEncoder(w)
		err := enc.Encode(struct {
			Hostname string
			Date     string
			Content  string
		}{
			Hostname: e.Name,
			Date:     e.Date.Format(file.DefaultDateFormat),
			Content:  e.Content.String(),
		})
		if err != nil {
			log.Printf("Error: %s", err)
			http.Error(w, "error :(", 500)
			return
		}
	} else {

		em := struct {
			Hostname string
			Date     string
			Content  string
		}{
			vars["hostname"],
			t.String(),
			e.Content.String(),
		}

		io.WriteString(w, mustache.RenderFileInLayout("templates/entry.html.mustache", "templates/layout.html.mustache", em))
	}
}

func showBackupDate(store configstore.Store, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	t, err := time.Parse(file.DefaultDateFormat, vars["date"])
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, "error :(", 500)
		return
	}

	datesStore, err := store.Dates(vars["hostname"])
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, "error :(", 500)
		return
	}

	// Find the backup on or after the specified date
	var idx int
	for i, date := range datesStore {
		if t.Before(date) {
			idx = i - 1
		}
	}

	if idx == -1 {
		idx = 0
	}

	e := configstore.Entry{
		Name: vars["hostname"],
		Date: datesStore[idx],
	}

	err = store.Get(&e)
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, "error :(", 500)
		return
	}

	em := struct {
		Hostname string
		Date     string
		Content  string
	}{
		vars["hostname"],
		datesStore[idx].String(),
		e.Content.String(),
	}

	io.WriteString(w, mustache.RenderFileInLayout("templates/entry.html.mustache", "templates/layout.html.mustache", em))
}

func diffBackup(store configstore.Store, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	t1, err := time.Parse(file.DefaultDateFormat, vars["date1"])
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, "error :(", 500)
		return
	}

	t2, err := time.Parse(file.DefaultDateFormat, vars["date2"])
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, "error :(", 500)
		return
	}

	e1 := configstore.Entry{
		Name: vars["hostname"],
		Date: t1,
	}

	e2 := configstore.Entry{
		Name: vars["hostname"],
		Date: t2,
	}

	err = store.Get(&e1)
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, "error :(", 500)
		return
	}

	err = store.Get(&e2)
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, "error :(", 500)
		return
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(e1.Content.String(), e2.Content.String(), true)

	em := struct {
		Hostname string
		Date1    time.Time
		Date2    time.Time
		Diff     string
	}{
		vars["hostname"],
		t1,
		t2,
		dmp.DiffPrettyHtml(diffs),
	}

	io.WriteString(w, mustache.RenderFileInLayout("templates/diff.html.mustache", "templates/layout.html.mustache", em))
}
