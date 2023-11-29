package main

import (
	"flag"
	"fmt"
	"net/http"
	"sync"
)

func main() {
	// Define the number of servers
	var numServers int
//	numServers := *flag.Int("n", 3 ,"num of server")
	flag.IntVar(&numServers, "n", 3, "number of demo servers to serve")
	flag.Parse()

	// Use a wait group to wait for all servers to start
	var wg sync.WaitGroup
	wg.Add(numServers)

	// Create and start multiple servers
	for i := 0; i < numServers; i++ {
		go func(port int) {
			defer wg.Done()

			// Create a handler function
			handler := func(w http.ResponseWriter, req *http.Request) {
				fmt.Fprintf(w, "Hello from Server %d!\n", port)
			}

			// Start the server on a unique port
			portStr := fmt.Sprintf(":808%d", port)
			fmt.Printf("Server %d listening on %s\n", port, portStr)
			http.HandleFunc("/" + portStr, handler)
			err := http.ListenAndServe(portStr, nil)
			if err != nil {
				panic(err)
			}
		}(i + 1) // Adding 1 to make port numbers unique
	}

	// Wait for all servers to start
	wg.Wait()
}

