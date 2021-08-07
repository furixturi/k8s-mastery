// main.go

package main

import (
	"fmt"
	"os"
)

func main() {
	var (
		SA_LOGIC_URL      = os.Getenv("SA_LOGIC_URL")
		SA_LOGIC_PORT     = os.Getenv("SA_LOGIC_PORT")
		saLogicServiceURI = SA_LOGIC_URL + ":" + SA_LOGIC_PORT + "/analyse/sentiment"
		SA_WEBAPP_PORT    = os.Getenv("SA_WEBAPP_PORT")
		// SA_WEBAPP_PORT    = "8080"
		// saLogicServiceURI = "http://localhost:5000/analyse/sentiment"
	)

	fmt.Printf("==== ENV ====\nSA_LOGIC_URL: %s\nSA_LOGIC_PORT: %s\nsaLogicServiceURI: %s\nSA_WEBAPP_PORT: %s\n",
		SA_LOGIC_URL, SA_LOGIC_PORT, saLogicServiceURI, SA_WEBAPP_PORT)

	a := App{}

	a.Initialize(saLogicServiceURI)
	a.Run(SA_WEBAPP_PORT)
}
