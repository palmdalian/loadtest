package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/palmdalian/loadtest"
)

func main() {
	// Command-line flags
	var url string
	var duration int
	var targetLatency int
	flag.StringVar(&url, "url", "", "The URL to test (required)")
	flag.IntVar(&duration, "duration", 10, "Duration of each test in seconds")
	flag.IntVar(&targetLatency, "target", 100, "Target latency (ms) for prediction")
	concurrencyLevels := flag.String("concurrency", "1,2,10,50,100,200", "Comma-separated list of concurrency levels")
	checkPrediction := flag.Bool("check", false, "Re-run apib to check prediction")
	plotFlag := flag.Bool("plot", false, "Generate plots (latency.png and rps.png)")
	flag.Parse()

	if url == "" {
		fmt.Println("Error: -url flag is required")
		flag.Usage()
		return
	}

	fmt.Printf("Starting load tests for URL: %s\n", url)

	// Parse concurrency levels
	concurrencyList := []int{}
	for _, level := range strings.Split(*concurrencyLevels, ",") {
		conc, err := strconv.Atoi(level)
		if err != nil {
			fmt.Printf("Invalid concurrency level: %s\n", level)
			flag.Usage()
			return
		}
		concurrencyList = append(concurrencyList, conc)
	}

	// Run load tests
	runner := loadtest.NewRunner(url, duration, targetLatency, loadtest.Latency90, concurrencyList, *checkPrediction, *plotFlag)
	predictedConcurrency, err := runner.Run()
	if err != nil {
		fmt.Printf("Error running load tests: %v\n", err)
		return
	}

	fmt.Printf("Predicted concurrency level: %.2f\n", predictedConcurrency)
	fmt.Println("Tests complete.")
}
