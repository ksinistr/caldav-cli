package main

import (
	"os"

	"github.com/ksinistr/caldav-cli/internal/cli"
)

func main() {
	os.Exit(cli.Run())
}
