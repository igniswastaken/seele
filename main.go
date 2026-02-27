package main

import (
	"flag"
	"fmt"

	"github.com/zerothy/seele/cmd"
	"github.com/zerothy/seele/service"
)

func main() {
	port := flag.String("port", "8080", "port to listen on")
	dataDir := flag.String("dir", ".", "directory to store data")
	isProxy := flag.Bool("proxy", false, "run as proxy")
	joinAddr := flag.String("join", "", "address of a node to join")
	flag.Parse()

	if *isProxy {
		if err := service.StartProxy(*port, *joinAddr); err != nil {
			fmt.Println("Error starting proxy:", err)
		}
		return
	}

	fmt.Printf("Starting Seele on :%s (data: %s)...\n", *port, *dataDir)
	if err := cmd.StartServer(*port, *dataDir, *joinAddr); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
