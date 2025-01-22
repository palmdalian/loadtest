package loadtest

import (
	"bytes"
	"fmt"
	"image/color"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

type Runner struct {
	URL               string
	Duration          int
	TargetLatency     int
	LatencyPercentile LatencyPercentile
	ConcurrencySteps  []int
	CheckPrediction   bool
	Plot              bool
}

func NewRunner(url string, duration, targetLatency int, latencyPercentile LatencyPercentile, concurrency []int, checkPrediction, plot bool) *Runner {
	return &Runner{
		URL:               url,
		Duration:          duration,
		TargetLatency:     targetLatency,
		LatencyPercentile: latencyPercentile,
		ConcurrencySteps:  concurrency,
		CheckPrediction:   checkPrediction,
		Plot:              plot,
	}
}

func (r *Runner) Run() (float64, error) {
	// Warmup request
	if err := runAPIBWarmup(r.URL); err != nil {
		return 0, fmt.Errorf("warmup failed: %w", err)
	}

	// Run tests for each concurrency level
	results := []*TestResult{}
	for _, concurrency := range r.ConcurrencySteps {
		// Make sure to drain connections between runs
		if err := waitForConnectionsToClear(100); err != nil {
			return 0, fmt.Errorf("failed to check existing connections: %w", err)
		}
		fmt.Printf("Running test with concurrency %d...\n", concurrency)
		result, err := runAPIB(concurrency, r.Duration, r.URL)
		if err != nil {
			fmt.Printf("Test failed for concurrency %d: %v\n", concurrency, err)
			continue
		}
		results = append(results, result)
		fmt.Println(result.Print(r.LatencyPercentile))
	}

	// Analyze and predict
	predictedConcurrency, err := analyzeAndPredict(r.TargetLatency, r.LatencyPercentile, results)
	if err != nil {
		return 0, fmt.Errorf("failed to analyze results: %w", err)
	}

	if r.CheckPrediction {
		rounded := int(math.Round(predictedConcurrency))
		fmt.Printf("Re-running tests to check predicted concurrency %f (rounded to %d)\n", predictedConcurrency, rounded)
		result, err := runAPIB(rounded, r.Duration, r.URL)
		if err != nil {
			return 0, fmt.Errorf("test failed for predicted concurrency %.2f: %w", predictedConcurrency, err)
		} else {
			results = append(results, result)
			fmt.Println(result.Print(r.LatencyPercentile))
		}

	}

	// Generate plots if requested
	if r.Plot {
		if err := plotResults(results, r.TargetLatency, r.LatencyPercentile); err != nil {
			return 0, err
		}
	}

	return predictedConcurrency, nil
}

func runAPIB(concurrency, duration int, url string) (*TestResult, error) {
	cmd := exec.Command("apib", "-S", "-c", fmt.Sprint(concurrency), "-d", fmt.Sprint(duration), url)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute apib: %w\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	if stderr.Len() > 0 {
		fmt.Println("Warning: apib produced stderr output:\n", stderr.String())
	}

	output := stdout.String()
	return parseCSVOutput(output)
}

func runAPIBWarmup(url string) error {
	cmd := exec.Command("apib", "-S", "-1", url)
	var out bytes.Buffer
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute apib: %w\nOutput: %s", err, out.String())
	}

	if stderr.Len() > 0 {
		return fmt.Errorf("apib produced stderr output: %s", stderr.String())
	}

	return nil
}

func waitForConnectionsToClear(threshold int) error {
	for {
		cmd := exec.Command("sh", "-c", "netstat -an | grep TIME_WAIT | wc -l")
		out, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to execute netstat: %w", err)
		}

		// Parse the number of TIME_WAIT connections
		count, err := strconv.Atoi(strings.TrimSpace(string(out)))
		if err != nil {
			return fmt.Errorf("failed to parse TIME_WAIT count: %w", err)
		}

		if count <= threshold {
			break
		}

		fmt.Printf("Waiting for connections to clear: %d TIME_WAIT\n", count)
		time.Sleep(5 * time.Second) // Adjust as needed
	}
	return nil
}

func plotResults(results []*TestResult, targetLatency int, latencyPercentile LatencyPercentile) error {
	// Prepare data points for plots
	performancePts := make(plotter.XYs, len(results))
	rpsPts := make(plotter.XYs, len(results))

	for i, res := range results {
		performancePts[i].X = float64(res.Connections)
		performancePts[i].Y = res.Latency(latencyPercentile)
		rpsPts[i].X = float64(res.Connections)
		rpsPts[i].Y = res.Throughput
	}

	// Perform quadratic regression
	a, b, c, err := quadraticRegression(results, latencyPercentile)
	if err != nil {
		return fmt.Errorf("failed to compute quadratic regression for plotting: %w", err)
	}

	// Generate prediction points for the quadratic fit
	numPredictionPoints := 100
	quadFitPts := make(plotter.XYs, numPredictionPoints)
	maxConcurrency := performancePts[len(performancePts)-1].X

	for i := 0; i < numPredictionPoints; i++ {
		x := maxConcurrency * float64(i) / float64(numPredictionPoints-1)
		y := a*x*x + b*x + c
		quadFitPts[i].X = x
		quadFitPts[i].Y = y
	}

	// Predict the concurrency at the target latency
	predictedConcurrency, err := predictConcurrencyQuad(a, b, c, targetLatency)
	if err != nil {
		return fmt.Errorf("failed to predict concurrency at target latency: %w", err)
	}

	// Plot the full Latency vs. Concurrency
	latencyPlot := plot.New()
	latencyPlot.Title.Text = "Latency vs. Concurrency"
	latencyPlot.X.Label.Text = "Concurrency"
	latencyPlot.Y.Label.Text = "Latency (ms)"

	// Add original data points
	dataLine, err := plotter.NewScatter(performancePts)
	if err != nil {
		return err
	}
	dataLine.GlyphStyle.Shape = draw.CircleGlyph{}

	// Add quadratic fit line
	quadLine, err := plotter.NewLine(quadFitPts)
	if err != nil {
		return err
	}
	quadLine.LineStyle.Width = vg.Points(2)
	quadLine.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255}

	latencyPlot.Add(dataLine, quadLine)
	latencyPlot.Legend.Add("Data Points", dataLine)
	latencyPlot.Legend.Add("Quadratic Fit", quadLine)

	// Save the full latency plot
	if err := latencyPlot.Save(6*vg.Inch, 4*vg.Inch, "latency_with_fit.png"); err != nil {
		return err
	}

	// Plot the zoomed Latency vs. Concurrency
	zoomedPlot := plot.New()
	zoomedPlot.Title.Text = "Zoomed: Latency vs. Concurrency"
	zoomedPlot.X.Label.Text = "Concurrency"
	zoomedPlot.Y.Label.Text = "Latency (ms)"

	xMin := predictedConcurrency * 0.9
	xMax := predictedConcurrency * 1.1

	// Set axis boundaries to focus on the region around the target latency
	zoomedPlot.X.Min = xMin // Adjust to add context around the prediction
	zoomedPlot.X.Max = xMax
	zoomedPlot.Y.Min = float64(targetLatency) - 50 // Adjust to add context around the target latency
	zoomedPlot.Y.Max = float64(targetLatency) + 50

	targetPoint, err := plotter.NewScatter(plotter.XYs{{X: predictedConcurrency, Y: float64(targetLatency)}})
	if err != nil {
		return err
	}
	targetPoint.GlyphStyle.Shape = draw.CircleGlyph{}

	// Generate prediction points for the quadratic fit
	predictedFitPts := make(plotter.XYs, numPredictionPoints)
	// Calculate the step size based on xMin and xMax
	step := (xMax - xMin) / float64(numPredictionPoints-1)
	for i := 0; i < numPredictionPoints; i++ {
		// Start directly at xMin and increment by the step
		x := xMin + step*float64(i)

		// Compute y using the quadratic equation
		y := a*x*x + b*x + c

		// Add the point to the slice
		predictedFitPts[i] = plotter.XY{X: x, Y: y}
	}

	// Add quadratic fit line
	predictedLine, err := plotter.NewLine(predictedFitPts)
	if err != nil {
		return err
	}
	predictedLine.LineStyle.Width = vg.Points(2)
	predictedLine.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255}

	// Add the same data and fit lines
	zoomedPlot.Add(targetPoint, predictedLine)
	zoomedPlot.Legend.Add("Data Points", dataLine)
	zoomedPlot.Legend.Add("Quadratic Fit", predictedLine)

	// Save the zoomed latency plot
	if err := zoomedPlot.Save(6*vg.Inch, 4*vg.Inch, "latency_with_fit_zoomed.png"); err != nil {
		return err
	}

	// Plot RPS vs. Concurrency
	rpsPlot := plot.New()
	rpsPlot.Title.Text = "RPS vs. Concurrency"
	rpsPlot.X.Label.Text = "Concurrency"
	rpsPlot.Y.Label.Text = "RPS"
	rpsLine, err := plotter.NewLine(rpsPts)
	if err != nil {
		return err
	}
	rpsPlot.Add(rpsLine)
	rpsPlot.Legend.Add("RPS Data", rpsLine)

	// Save the RPS plot
	if err := rpsPlot.Save(6*vg.Inch, 4*vg.Inch, "rps.png"); err != nil {
		return err
	}

	fmt.Println("Plots generated: latency_with_fit.png, latency_with_fit_zoomed.png, and rps.png")
	return nil
}
