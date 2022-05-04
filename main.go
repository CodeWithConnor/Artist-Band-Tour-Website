package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"sync"
)

type bandInfo struct {
	artists   []artist
	locations []locations
	dates     []dates
	relations []relations
}

type artist struct {
	Id           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
}

type loc struct {
	Index []locations `json:"index"`
}

type locations struct {
	Id        int      `json:"id"`
	Locations []string `json:"locations"`
}

type dat struct {
	Index []dates `json:"index"`
}

type dates struct {
	Id    int      `json:"id"`
	Dates []string `json:"dates"`
}

type rel struct {
	Index []relations `json:"index"`
}

type relations struct {
	Id             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

// For handling custom 404 response
type NotFoundRedirectRespWr struct {
	http.ResponseWriter // We embed http.ResponseWriter
	status              int
}

func (w *NotFoundRedirectRespWr) WriteHeader(status int) {
	w.status = status // Store the status for our own use
	if status != http.StatusNotFound {
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *NotFoundRedirectRespWr) Write(p []byte) (int, error) {
	if w.status != http.StatusNotFound {
		return w.ResponseWriter.Write(p)
	}
	return len(p), nil // Lie that we successfully written it
}

func artistUnmarshal(apiWait *sync.WaitGroup) []artist {
	url := "https://groupietrackers.herokuapp.com/api/artists"

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}

	body, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		panic(err.Error())
	}
	var artists []artist
	err3 := json.Unmarshal(body, &artists)
	if err3 != nil {
		fmt.Println("whoops:", err3)
	}
	apiWait.Done()
	return artists
}

func locationUnmarshal(apiWait *sync.WaitGroup) []locations {
	url := "https://groupietrackers.herokuapp.com/api/locations"

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}

	body, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		panic(err.Error())
	}
	var loc loc
	err3 := json.Unmarshal(body, &loc)
	if err3 != nil {
		fmt.Println("whoops:", err3)
	}
	apiWait.Done()
	return loc.Index
}

func datesUnmarshal(apiWait *sync.WaitGroup) []dates {
	url := "https://groupietrackers.herokuapp.com/api/dates"

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}

	body, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		panic(err.Error())
	}
	var dat dat
	err3 := json.Unmarshal(body, &dat)
	if err3 != nil {
		fmt.Println("whoops:", err3)
	}
	apiWait.Done()
	return dat.Index
}

func relationUnmarshal(apiWait *sync.WaitGroup) []relations {
	url := "https://groupietrackers.herokuapp.com/api/relation"

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}

	body, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		panic(err.Error())
	}
	var rel rel
	err3 := json.Unmarshal(body, &rel)
	if err3 != nil {
		fmt.Println("whoops:", err3)
	}
	apiWait.Done()
	return rel.Index
}

func getAttr(obj interface{}, fieldName string) reflect.Value {
	pointToStruct := reflect.ValueOf(obj) // addressable
	curStruct := pointToStruct.Elem()
	if curStruct.Kind() != reflect.Struct {
		panic("not struct")
	}
	curField := curStruct.FieldByName(fieldName) // type: reflect.Value
	if !curField.IsValid() {
		panic("not found:" + fieldName)
	}
	return curField
}

func main() {
	fs := wrapHandler(http.FileServer(http.Dir("./static")))
	http.HandleFunc("/", fs)
	// when the user clicks 'Show Artists' button inside of index.html, it calls portal()
	http.HandleFunc("/portal", portal)

	log.Println("Listening on :3000...")
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func wrapHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nfrw := &NotFoundRedirectRespWr{ResponseWriter: w}
		h.ServeHTTP(nfrw, r)
		if nfrw.status == 404 {
			log.Printf("Redirecting %s to 404.html.", r.RequestURI)
			http.Redirect(w, r, "/404.html", http.StatusFound)
		}
	}
}

func portal(w http.ResponseWriter, r *http.Request) {
	var bandInfo bandInfo
	var wait sync.WaitGroup

	wait.Add(4)
	bandInfo.artists = artistUnmarshal(&wait)
	bandInfo.locations = locationUnmarshal(&wait)
	bandInfo.dates = datesUnmarshal(&wait)
	bandInfo.relations = relationUnmarshal(&wait)
	wait.Wait()

	// redirects user to /portal.html
	temp, err := template.ParseFiles("static/portal.html")
	if err != nil {
		fmt.Fprintf(w, http.StatusText(404))
	}

	temp.Execute(w, nil)

	// here we're using FprintF to print each individual card, containing each artist/band's data to our portal.html page
	// we range through each artist, creating a new card each time (in turn, populating the page with as many cards as required)
	for i := range bandInfo.artists {
		fmt.Fprintf(w, "<div class=\"card col-md-4\"><div class=\"card-title\"><h2>")

		// Artist Name
		fmt.Fprintf(w, bandInfo.artists[i].Name)

		// First Album
		fmt.Fprintf(w, "<small> First album: "+bandInfo.artists[i].FirstAlbum+"</small></h2>")

		// Artist Image
		fmt.Fprintf(w, "<img src='"+bandInfo.artists[i].Image+"' width=\"100%\">")

		// Members
		fmt.Fprintf(w, "</div><div class=\"card-flap flap1\"><div class=\"card-description\"><ul class=\"task-list\"><div class=\"members\"><div class=\"members-title\">Members</div><div class=\"members-item\">")

		// For each member in members
		for _, member := range bandInfo.artists[i].Members {
			fmt.Fprintf(w, "<div class=\"members-item\">")
			// Print member into 'members-item' div
			fmt.Fprintf(w, member)
			fmt.Fprintf(w, "</div>")
		}

		// Creation Date
		fmt.Fprintf(w, "</div><div class=\"members\"><div class=\"members-title\">Creation Date</div><div class=\"members-item\">")
		fmt.Fprintf(w, "%v", bandInfo.artists[i].CreationDate)
		fmt.Fprintf(w, "</div></div>")

		// Locations/Dates
		fmt.Fprintf(w, "<div class=\"locations-dates\"><div class=\"members\"><div class=\"members-title\">Locations/Dates</div>")

		for _, location := range bandInfo.locations[i].Locations {
			fmt.Fprintf(w, "<div class=\"location-item\">")
			fmt.Fprintf(w, location)
			fmt.Fprintf(w, "</div>")
		}

		for _, date := range bandInfo.dates[i].Dates {
			fmt.Fprintf(w, "<div class=\"date-item\">")
			fmt.Fprintf(w, date)
			fmt.Fprintf(w, "</div>")
		}

		fmt.Fprintf(w, "</div></div>")

		// Close Button
		fmt.Fprintf(w, "</ul></div>")

		// Date Explanation Div
		fmt.Fprintf(w, "<div class=\"card-flap flap2\"><div class=\"card-actions\"><a class=\"btn\" href=\"#\">Close</a></div></div></div></div>")
	}

	// End HTML file
	fmt.Fprintf(w, "</div><script src='https://cdnjs.cloudflare.com/ajax/libs/jquery/3.1.1/jquery.min.js'></script><script  src=\"./script.js\"></script></body></html>")
}
