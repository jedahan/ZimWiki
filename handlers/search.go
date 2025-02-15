package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
	"strconv"
	"math"

	"github.com/JojiiOfficial/ZimWiki/zim"
	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"
)

func searchSingle(query string, nbResultsPerPage int, resultsUntil int, wiki *zim.File) ([]zim.SRes, int, int) {
	// Search entries
	entries := wiki.SearchForEntry(query)
	// Sort them by similarity
	sort.Sort(sort.Reverse(zim.ByPercentage(entries)))

	// Calculate the first result displayed in the page
	resultsStart := resultsUntil - nbResultsPerPage

	// If the result of the calculation is negative
	if resultsStart < 0 {
		resultsStart = 0
	}

	// If the result of the calculation is greater than the total of the results
	if resultsStart > len(entries) {
		resultsStart = len(entries)
	}

	// If the number of results to be displayed per page is lower than the total number of results
	if len(entries) < nbResultsPerPage {
		resultsUntil = len(entries)
	}

	// If the result to be the maximum result of the page is greater than the number of total results
	if resultsUntil > len(entries) {
		resultsUntil = len(entries)
	}

	// Calculate the total number of pages
	nbPages := int(math.Ceil(float64(len(entries)) / float64(nbResultsPerPage)))

	return entries[resultsStart:resultsUntil], len(entries), nbPages
}

// Search in all available wikis
func searchGlobal(query string, nbResultsPerPage int, resultsUntil int, handler *zim.Handler) ([]zim.SRes, int, int) {
	var results []zim.SRes
	mx := sync.Mutex{}
	wg := sync.WaitGroup{}
	files := handler.GetFiles()

	wg.Add(len(files))

	// Concurrent global search
	for i := range files {
		go func(index int) {
			// Search
			r := files[index].SearchForEntry(query)

			// Append results
			mx.Lock()
			results = append(results, r...)
			mx.Unlock()

			wg.Done()
		}(i)
	}

	wg.Wait()

	// Sort by similarity
	sort.Sort(sort.Reverse(zim.ByPercentage(results)))

	resultsStart := resultsUntil - nbResultsPerPage

	// If the result of the calculation is negative, set the variable to 0
	if resultsStart < 0 {
		resultsStart = 0
	}

	// If the result of the calculation is greater than the total of the results
	if resultsStart > len(results) {
		resultsStart = len(results)
	}

	// If the number of results to be displayed per page is lower than the total number of results
	if len(results) < nbResultsPerPage {
		resultsUntil = len(results)
	}

	// If the result to be the maximum result of the page is greater than the number of total results
	if resultsUntil > len(results) {
		resultsUntil = len(results)
	}

	// Calculate the total number of pages
	nbPages := int(math.Ceil(float64(len(results)) / float64(nbResultsPerPage)))

	return results[resultsStart:resultsUntil], len(results), nbPages
}

// Search handles serach requests
func Search(w http.ResponseWriter, r *http.Request, hd HandlerData) error {
	vars := mux.Vars(r)
	wiki, ok := vars["wiki"]
	if !ok {
		return fmt.Errorf("Missing parameter")
	}
	var query string
	var actualPageNb int

	if r.Method == "POST" {
		// Get Post search query
		query = r.PostFormValue("sQuery")
		// Get the number of the current page
		actualPageNb, _ = strconv.Atoi(r.PostFormValue("pageNumber"))
	}

	if len(query) == 0 {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
		return nil
	}

	// If the variable is not defined or negative, we set it to 1
	if (actualPageNb == 0) || (actualPageNb < 0) {
		actualPageNb = 1
	}

	// The number of results displayed per page
	nbResultsPerPage := 12

	// A small calculus to know the last result to display
	resultsUntil := nbResultsPerPage * actualPageNb

	var res []zim.SRes
	var source string
	var nbResults int
	var nbPages int

	start := time.Now()
	if wiki == "-" {
		source = "global search"

		res, nbResults, nbPages = searchGlobal(query, nbResultsPerPage, resultsUntil, hd.ZimService)
	} else {
		// Wiki search
		z := hd.ZimService.FindWikiFile(wiki)
		if z == nil {
			http.NotFound(w, r)
			return ErrNotFound
		}
		source = z.File.Title()

		// Search for query in WIKI
		res, nbResults, nbPages = searchSingle(query, nbResultsPerPage, resultsUntil, z)
	}

	var previousPage int
	var nextPage int

	if actualPageNb != 1 {
		previousPage = actualPageNb - 1
	}

	if  nbResults - (nbResultsPerPage * (actualPageNb)) > 0 {
		nextPage = actualPageNb + 1
	}

	favCache := make(map[string]string)
	results := make([]SearchResult, len(res))

	for i := range res {
		// Get Favicon of each file
		fav, has := favCache[res[i].GetID()]
		if !has {
			favIcon, err := res[i].Favicon()
			if err == nil {
				fav = zim.GetRawWikiURL(res[i].File, favIcon)
			} else {
				fav = ""
			}

			favCache[res[i].GetID()] = fav
		}

		results[i] = SearchResult{
			Img:   fav,
			Link:  zim.GetWikiURL(res[i].File, *res[i].DirectoryEntry),
			Title: string(res[i].Title()),
			Wiki:  wiki,
		}
	}

	timeTook := time.Since(start)

	var resultText string

	if (nbResults == 0) {
		resultText = "No result"
	} else if (nbResults == 1) {
		resultText = "1 result"
	} else {
		resultText = strconv.Itoa(nbResults) + " results"
	}

	log.Info(resultText, " in ", timeTook)

	// Redirect to wiki page if only
	// one search result was found
	if len(results) == 1 {
		http.Redirect(w, r, results[0].Link, http.StatusMovedPermanently)
		return nil
	}

	return serveTemplate(SearchTemplate, w, r, TemplateData{
		Wiki: wiki,
		SearchTemplateData: SearchTemplateData{
			Results:      results,
			QueryText:    query,
			SearchSource: source,
			NbResults:    nbResults,
			TimeTook:     timeTook,
			ActualPageNb: actualPageNb,
			NbPages:      nbPages,
			PreviousPage: previousPage,
			NextPage:     nextPage,
		},
	})
}
