// app.go

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type App struct {
	Router      *mux.Router
	service_url string
}

func (a *App) Initialize(service_url string) {
	a.service_url = service_url
	a.Router = mux.NewRouter()
	a.initializeRoutes()
}

func (a *App) Run(port string) {
	c := cors.New(cors.Options{
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodHead,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"*",
		},
	})
	fmt.Printf("App running on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handlers.LoggingHandler(os.Stdout, c.Handler(a.Router))))
}

func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/healthcheck", a.healthCheck).Methods("GET")
	a.Router.HandleFunc("/sentiment", a.retrieveSentiment).Methods("POST")
}

func (a *App) healthCheck(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{"result": "ok"})
}

// reference: https://zetcode.com/golang/getpostrequest/
func (a *App) retrieveSentiment(w http.ResponseWriter, r *http.Request) {
	// get the request body {"sentence": "..."}
	var reqBody map[string]interface{}
	json.NewDecoder(r.Body).Decode(&reqBody)
	fmt.Printf("==== frontend request body ====\n%v\n", reqBody)

	// prep external service request body
	json_data, err := json.Marshal(reqBody)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Cannot parse the sentence")
		log.Printf("Cannot parse sentence from sa-frontend\n%v\n", err)
	} else {
		// post to external service to get polarity
		resp, err := http.Post(a.service_url, "application/json", bytes.NewBuffer(json_data))

		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Cannot get sentiment result for the sentence")
			log.Printf("Cannot get response from sa-logic:\n%v\n", err)
		} else {
			// prep response {"sentence":"...", "polarity":"..."}
			var resBody map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&resBody)
			fmt.Printf("==== sentiment service response ====\n%v\n", resBody)

			// send response
			respondWithJSON(w, http.StatusOK, resBody)
		}
	}
}

/** helper functions **/
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
