package main

import "github.com/insightsengineering/scribe/cmd"

// TODO this has to be replaced with actual checking whether we're running in a pipeline
const Interactive = false

func main() {
	cmd.Execute()
}
