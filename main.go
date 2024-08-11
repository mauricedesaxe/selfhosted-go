package main

import (
	"go-on-rails/cmd"
)

func main() {
	cmd.Run()
	// Block the main goroutine to keep the server running
	select {}
}
