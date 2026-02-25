// Judge Dispatcher — Kafka consumer that spawns K8s Jobs for grading submissions.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("judge-dispatcher starting...")
	os.Exit(0)
}
