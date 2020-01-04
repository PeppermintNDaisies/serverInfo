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
	"github.com/google/uuid"
	"github.com/goquery"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

var router *chi.Mux
var db *sql.DB

// Post type details
type Items struct {
	Items []ServerInfo `json:"items"`
}

type ServerInfo struct {
	Domain string `json:"domain"`
	Info   Info   `json:"info"`
}

type Info struct {
	Servers          []Server  `json:"servers"`
	ServersChanged   bool      `json:"servers_changed"`
	SSLGrade         string    `json:"ssl_grade"`
	PreviousSSLGrade string    `json:"previous_ssl_grade"`
	Logo             string    `json:"logo"`
	Title            string    `json:"title"`
	IsDown           bool      `json:"is_down"`
	Created          time.Time `json:"created_at"`
	Updated          time.Time `json:"updated_at"`
}

type ServerDB struct {
	Id       uuid.UUID `json:"id"`
	Address  string    `json:"address"`
	SSLGrade string    `json:"ssl_grade"`
	Country  string    `json:"country"`
	Owner    string    `json:"owner"`
}

type Server struct {
	Address  string `json:"address"`
	SSLGrade string `json:"ssl_grade"`
	Country  string `json:"country"`
	Owner    string `json:"owner"`
}

var Grades = [7]string{"A+", "A", "B", "C", "D", "E", "F"}

func init() {
	router = chi.NewRouter()
	router.Use(middleware.Recoverer)

	var err error

	db, err = sql.Open("postgres",
		"postgresql://maxroach@localhost:26257/servers?ssl=true&sslmode=require&sslrootcert=certs/ca.crt&sslkey=certs/client.maxroach.key&sslcert=certs/client.maxroach.crt")
	catch(err)
}

func routers() *chi.Mux {

	router.Get("/servers", AllServers)
	router.Get("/serverinfo/{domain}", GetServerInfo)

	return router
}

//-------------- API ENDPOINT ------------------//

// CreatePost create a new post
func AllServers(w http.ResponseWriter, r *http.Request) {

	var response []ServerInfo

	rows, err := db.Query(
		"SELECT id, domain, servers_changed, ssl_grade, previous_ssl_grade, logo, title, is_down, created, updated FROM serverInfo")
	catch(err)

	defer rows.Close()

	for rows.Next() {

		var responseInfo Info
		var serverInfo ServerInfo
		var servers []Server

		// Print out the returned values.
		var id uuid.UUID
		var domain, sslGradeDB string
		var serversChangedDB sql.NullBool
		var previousSslGradeDB, logo, title sql.NullString
		var isDown bool
		var createdAtDB time.Time
		var updatedAtDB pq.NullTime

		if err := rows.Scan(&id, &domain, &serversChangedDB, &sslGradeDB, &previousSslGradeDB, &logo, &title, &isDown, &createdAtDB, &updatedAtDB); err != nil {
			log.Print(err)
		} else {

			serverInfo.Domain = domain

			responseInfo.SSLGrade = sslGradeDB
			if previousSslGradeDB.Valid {
				responseInfo.PreviousSSLGrade = previousSslGradeDB.String
			}
			if logo.Valid {
				responseInfo.Logo = logo.String
			}
			if title.Valid {
				responseInfo.Title = title.String
			}
			responseInfo.IsDown = isDown
			responseInfo.Created = createdAtDB
			if updatedAtDB.Valid {
				responseInfo.Updated = updatedAtDB.Time
			}

			rows, err := db.Query("Select server_id From servers_info WHERE info_id=$1 ", id)
			catch(err)

			defer rows.Close()

			for rows.Next() {
				var serverID uuid.UUID

				er := rows.Scan(&serverID)

				if er != nil {
					log.Print(er)
				}

				var server Server

				row := db.QueryRow(
					"SELECT id, ipaddress, ssl_grade, country, owner FROM servers WHERE id=$1 ", serverID)

				// Print out the returned values.
				var id uuid.UUID
				var addressDB, sslGradeDB, country, owner string

				if err := row.Scan(&id, &addressDB, &sslGradeDB, &country, &owner); err != nil {
					log.Print(err)
				}

				if row != nil {
					server.Address = addressDB
					server.SSLGrade = sslGradeDB
					server.Country = country
					server.Owner = owner
				}

				servers = append(servers, server)

			}
			responseInfo.Servers = servers
			serverInfo.Info = responseInfo

		}
		response = append(response, serverInfo)
	}

	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(response)
}

// Get server information from domain entered by user
func GetServerInfo(w http.ResponseWriter, r *http.Request) {

	domain := chi.URLParam(r, "domain")

	row := db.QueryRow(
		"SELECT id, servers_changed, ssl_grade, previous_ssl_grade, logo, title, is_down, created, updated FROM serverInfo WHERE domain=$1 ", domain)

	var response Info
	var servers []Server

	// Print out the returned values.
	var id uuid.UUID
	var serversChangedDB sql.NullBool
	var sslGradeDB string
	var previousSslGradeDB, logo, title sql.NullString
	var isDown bool
	var createdAtDB time.Time
	var updatedAtDB pq.NullTime

	if err := row.Scan(&id, &serversChangedDB, &sslGradeDB, &previousSslGradeDB, &logo, &title, &isDown, &createdAtDB, &updatedAtDB); err != nil {
		response = searchInfo(domain, response)
	} else {

		response.SSLGrade = sslGradeDB
		if previousSslGradeDB.Valid {
			response.PreviousSSLGrade = previousSslGradeDB.String
		}
		if logo.Valid {
			response.Logo = logo.String
		}
		if title.Valid {
			response.Title = title.String
		}
		response.IsDown = isDown
		response.Created = createdAtDB
		if updatedAtDB.Valid {
			response.Updated = updatedAtDB.Time
		}

		rows, err := db.Query("Select server_id From servers_info WHERE info_id=$1 ", id)
		catch(err)

		defer rows.Close()

		for rows.Next() {
			var serverID uuid.UUID

			er := rows.Scan(&serverID)

			if er != nil {
				log.Print(er)
			}

			var server Server

			row := db.QueryRow(
				"SELECT id, ipaddress, ssl_grade, country, owner FROM servers WHERE id=$1 ", serverID)

			// Print out the returned values.
			var id uuid.UUID
			var addressDB, sslGradeDB, country, owner string

			if err := row.Scan(&id, &addressDB, &sslGradeDB, &country, &owner); err != nil {
				log.Print(err)
			}

			if row != nil {
				server.Address = addressDB
				server.SSLGrade = sslGradeDB
				server.Country = country
				server.Owner = owner
			}

			servers = append(servers, server)

		}

		response.Servers = servers
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

}

func main() {
	routers()
	http.ListenAndServe(":8005", Logger())
}

func searchInfo(domain string, response Info) Info {
	log.Print("Search")
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

	status := masnun["status"].(string)

	for !strings.EqualFold(status, "READY") {
		resp, err = http.Get(req)
		if err != nil {
			// handle error
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}

		json.Unmarshal(body, &masnun)

		status = masnun["status"].(string)
	}

	var endpoints []interface{}
	endpoints = masnun["endpoints"].([]interface{})

	servers := make([]Server, len(endpoints))
	serverIDs := make([]uuid.UUID, 0)

	sslGrade := "A+"

	for i := 0; i < len(endpoints); i++ {
		statusMessage := endpoints[i].(map[string]interface{})["statusMessage"].(string)

		for !strings.EqualFold(statusMessage, "Ready") {
			resp, err = http.Get(req)
			if err != nil {
				// handle error
			}
			defer resp.Body.Close()

			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatalln(err)
			}

			json.Unmarshal(body, &masnun)

			statusMessage = endpoints[i].(map[string]interface{})["statusMessage"].(string)
		}

		grade := endpoints[i].(map[string]interface{})["grade"]
		ipAddress := endpoints[i].(map[string]interface{})["ipAddress"]

		if (strings.EqualFold(sslGrade, "A+") && strings.Compare(grade.(string), sslGrade) < 0) ||
			(strings.Compare(grade.(string), sslGrade) > 0) {
			sslGrade = grade.(string)
		}
		owner := "Not found"
		country := "Not found"

		cmd := "whois " + ipAddress.(string) + " | grep OrgName"
		out, err := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			cmd := "whois " + ipAddress.(string) + " | grep org-name"
			out, err = exec.Command("bash", "-c", cmd).Output()
			if err != nil {
				cmd := "whois " + ipAddress.(string) + " | grep organisation"
				out, err = exec.Command("bash", "-c", cmd).Output()
				if err != nil {
					//
				}
			}
		}

		ownerArr := strings.Split(string(out), ":")
		owner = strings.ReplaceAll(ownerArr[1], " ", "")
		owner = strings.ReplaceAll(owner, "\n", "")

		cmd = "whois " + ipAddress.(string) + " | grep Country"
		out, err = exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			cmd = "whois " + ipAddress.(string) + " | grep country"
			out, err = exec.Command("bash", "-c", cmd).Output()
			if err != nil {
				//
			}
		}

		countryArr := strings.Split(string(out), ":")
		country = strings.ReplaceAll(countryArr[1], " ", "")
		country = strings.ReplaceAll(country, "\n", "")

		if strings.EqualFold(owner, "RIPENCC") {
			owner = "INFORMATION NOT AVAILABLE"
		}

		id, err := uuid.NewUUID()
		if err != nil {
			log.Print(err)
		}
		serverIDs = append(serverIDs, id)

		log.Print("id created ", id)
		log.Print(serverIDs)
		server := Server{
			Address:  ipAddress.(string),
			SSLGrade: grade.(string),
			Owner:    owner,
			Country:  country,
		}

		if _, err := db.Query(
			"INSERT INTO servers (ipAddress, ssl_grade, country, owner, id) "+
				"VALUES ($1, $2, $3, $4, $5) ",
			server.Address, server.SSLGrade, server.Country, server.Owner, id); err != nil {
			log.Fatal(err)
		}

		servers[i] = server
	}

	response.Servers = servers
	response.SSLGrade = sslGrade

	req = fmt.Sprintf("https://%s", domain)

	client := &http.Client{}
	newReq, err := http.NewRequest("GET", req, nil)

	newReq.Header.Add("User-Agent", "PostmanRuntime/7.21.0")
	resp, err = client.Do(newReq)

	if err != nil {
		// handle error
	}
	defer resp.Body.Close()

	fmt.Print(req)
	if resp.Status == "503" {
		response.IsDown = true
	} else {
		response.IsDown = false
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Find the review items
	doc.Find("title").EachWithBreak(func(i int, s *goquery.Selection) bool {
		// For each item found, get the band and title
		fmt.Print(s.Text())
		response.Title = s.Text()
		return false
	})

	doc.Find("link").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		attr, existsAttr := s.Attr("type")

		if existsAttr && strings.Contains(attr, "image/x-icon") {
			logo, existsLogo := s.Attr("href")
			if existsLogo {
				response.Logo = logo
				return
			}
		}

		if !existsAttr {
			attr, existsAttr = s.Attr("rel")

			if existsAttr && strings.Contains(attr, "icon") {
				logo, existsLogo := s.Attr("href")
				if existsLogo {
					response.Logo = logo
					return
				}
			}
		}

	})

	response.Created = time.Now()

	id, err := uuid.NewUUID()
	if err != nil {
		log.Print(err)
	}

	if _, err := db.Query(
		"INSERT INTO serverInfo (domain, ssl_grade, logo, title, is_down, created, id ) "+
			"VALUES ($1, $2, $3, $4, $5, $6, $7) ",
		domain, response.SSLGrade, response.Logo, response.Title, response.IsDown, time.Now(), id); err != nil {
		log.Fatal(err)
	}

	for _, element := range serverIDs {
		log.Print("id insert ", element)

		if err != nil {
			log.Print(err)
		}
		if _, err := db.Query(
			"INSERT INTO servers_info (id, info_id, server_id ) "+
				"VALUES (DEFAULT, $1, $2) ",
			id, element); err != nil {
			log.Fatal(err)
		}
	}

	return response
}
