package main

import (
	"log"
	"os"

	"github.com/codingpa-ws/foxbox/internal/cli"
)

func main() {
	err := cli.Start(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}
