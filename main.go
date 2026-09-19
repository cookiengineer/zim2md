// Command zim2md converts the HTML pages of an OpenZIM archive into
// Reader-Mode-style Markdown files.
package main

import (
	"os"

	"github.com/cookiengineer/zim2md/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
