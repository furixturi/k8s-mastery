// main.go

package main

import (
	"fmt"
)

func main() {
	var (
		// SA_LOGIC_URL   = os.Getenv("SA_LOGIC_URL")
		// SA_WEBAPP_PORT = os.Getenv("SA_WEBAPP_PORT")
		SA_LOGIC_URL   = "http://localhost:5000/analyse/sentiment"
		SA_WEBAPP_PORT = "8080"
	)

	fmt.Printf("SA_LOGIC_URL: %s\nSA_WEBAPP_PORT: %s\n", SA_LOGIC_URL, SA_WEBAPP_PORT)

	a := App{}

	a.Initialize(SA_LOGIC_URL)
	a.Run(SA_WEBAPP_PORT)
}
