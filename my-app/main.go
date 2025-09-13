package main

import (
	"github.com/tristendillon/conduit/core/server"
)

func main() {
	server := server.NewServer()
	server.Start()
}
