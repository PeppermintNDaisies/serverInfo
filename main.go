package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	_ "github.com/lib/pq"
)

var router *chi.Mux
var db *sql.DB

// Post type details
type Info struct {
	Servers          []Server  `json:"servers"`
	ServersChanged   string    `json:"servers_changed"`
	SSLGrade         string    `json:"ssl_grade"`
	PreviousSSLGrade string    `json:"previous_ssl_grade"`
	Logo             string    `json:"logo"`
	Title            string    `json:"title"`
	IsDown           string    `json:"is_down"`
	Created          time.Time `json:"created_at"`
}
type Server struct {
	Address  string `json:"address"`
	SSLGrade string `json:"ssl_grade"`
	Country  string `json:"country"`
	Owner    string `json:"owner"`
}

type SSLLabs struct {
	Endpoints []json.RawMessage `json:"endpoints"`
}

func init() {
	router = chi.NewRouter()
	router.Use(middleware.Recoverer)

	var err error

	db, err = sql.Open("postgres",
		"postgresql://maxroach@localhost:26257/defaultdb?ssl=true&sslmode=require&sslrootcert=certs/ca.crt&sslkey=certs/client.maxroach.key&sslcert=certs/client.maxroach.crt")
	catch(err)
}

func routers() *chi.Mux {

	router.Get("/servers", DetailPost)
	router.Get("/serverinfo/{domain}", ServerInfo)

	return router
}

//-------------- API ENDPOINT ------------------//

// CreatePost create a new post
func DetailPost(w http.ResponseWriter, r *http.Request) {

	var post Info
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Kindly enter data with the event title and description only in order to update")
	}

	json.Unmarshal(reqBody, &post)
	log.Print(post)
	if _, err := db.Exec(
		"INSERT INTO posts (title, content) VALUES ($1, $2)", post.Title, post.Logo); err != nil {
		log.Fatal(err)
	}

	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(post)
}

// Get server information from domain entered by user
func ServerInfo(w http.ResponseWriter, r *http.Request) {

	domain := chi.URLParam(r, "domain")

	req := fmt.Sprintf("https://api.ssllabs.com/api/v3/analyze?host=%s", domain)

	resp, err := http.Get(req)
	if err != nil {
		// handle error
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var masnun map[string]interface{}
	json.Unmarshal(body, &masnun)

	var endpoints []interface{}
	endpoints = masnun["endpoints"].([]interface{})

	servers := make([]Server, len(endpoints))

	for i := 0; i < len(endpoints); i++ {
		grade := endpoints[i].(map[string]interface{})["grade"]
		ipAddress := endpoints[i].(map[string]interface{})["ipAddress"]

		cmd := "whois " + ipAddress.(string) + " | grep OrgName"
		out, err := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			log.Fatalln(err)
		}

		ownerArr := strings.Split(string(out), ":")
		owner := strings.ReplaceAll(ownerArr[1], " ", "")
		owner = strings.ReplaceAll(owner, "\n", "")
		log.Print(owner)

		cmd = "whois " + ipAddress.(string) + " | grep Country"
		out, err = exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			log.Fatalln(err)
		}

		countryArr := strings.Split(string(out), ":")
		country := strings.ReplaceAll(countryArr[1], " ", "")
		country = strings.ReplaceAll(country, "\n", "")
		log.Print(country)

		server := Server{
			Address:  ipAddress.(string),
			SSLGrade: grade.(string),
			Owner:    owner,
			Country:  country,
		}

		servers[i] = server
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}

func main() {
	routers()
	http.ListenAndServe(":8005", Logger())
}
