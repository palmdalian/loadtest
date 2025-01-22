package loadtest

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
)

func quadraticRegression(results []*TestResult, latencyPercentile LatencyPercentile) (float64, float64, float64, error) {
	// Extract concurrency and latency data
	n := len(results)
	if n < 3 {
		return 0, 0, 0, fmt.Errorf("insufficient data points for quadratic regression (need at least 3)")
	}

	concurrency := make([]float64, n)
	latency := make([]float64, n)
	for i, res := range results {
		if res.Errors > 0 {
			fmt.Printf("Warning: %d out of %d requests returned errors. Results may not be accurate\n", res.Errors, res.Completed)
		}
		concurrency[i] = float64(res.Connections)
		latency[i] = res.Latency(latencyPercentile)
	}

	// Create the design matrix for quadratic regression
	X := mat.NewDense(n, 3, nil) // Each row: [1, x, x^2]
	y := mat.NewVecDense(n, latency)
	for i := 0; i < n; i++ {
		X.Set(i, 0, 1)                             // Intercept term
		X.Set(i, 1, concurrency[i])                // Linear term
		X.Set(i, 2, concurrency[i]*concurrency[i]) // Quadratic term
	}

	// Compute (XᵀX)
	var XT mat.Dense
	XT.Mul(X.T(), X) // XT = XᵀX

	// Invert (XᵀX)
	var XTInv mat.Dense
	err := XTInv.Inverse(&XT)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to invert matrix: %w", err)
	}

	// Compute (Xᵀy)
	var XTy mat.VecDense
	XTy.MulVec(X.T(), y)

	// Compute beta = (XᵀX)^(-1)(Xᵀy)
	var beta mat.VecDense
	beta.MulVec(&XTInv, &XTy)

	// Extract coefficients: beta = [c, b, a]
	c := beta.AtVec(0) // Intercept
	b := beta.AtVec(1) // Linear coefficient
	a := beta.AtVec(2) // Quadratic coefficient

	return a, b, c, nil
}

func predictConcurrencyQuad(a, b, c float64, targetLatency int) (float64, error) {
	// Solve ax^2 + bx + c = targetLatency for x
	// x = (-b ± sqrt(b^2 - 4ac)) / 2a
	delta := b*b - 4*a*(c-float64(targetLatency))
	if delta < 0 {
		return 0, fmt.Errorf("no real solutions for concurrency at target latency %dms", targetLatency)
	}

	// Return the positive root (the other root is typically negative and irrelevant)
	return (-b + math.Sqrt(delta)) / (2 * a), nil
}

func validatePrediction(a, b, c float64, targetLatency int, latencyPercentile LatencyPercentile, results []*TestResult) error {
	// Calculate the discriminant
	discriminant := b*b - 4*a*(c-float64(targetLatency))

	if discriminant < 0 {
		// No real roots
		return fmt.Errorf("target latency %dms cannot be achieved based on the quadratic fit", targetLatency)
	}

	// Calculate the roots
	root1 := (-b + math.Sqrt(discriminant)) / (2 * a)
	root2 := (-b - math.Sqrt(discriminant)) / (2 * a)

	// Find the observed range
	minConnections := float64(results[0].Connections)
	maxConnections := float64(results[len(results)-1].Connections)

	// Check if the roots are within the observed range
	if (root1 < minConnections || root1 > maxConnections) && (root2 < minConnections || root2 > maxConnections) {
		return fmt.Errorf("predicted roots (%.2f, %.2f) are outside the observed concurrency range [%.0f, %.0f]. Target latency may not be valid", root1, root2, minConnections, maxConnections)
	}

	// Check if all observed latencies are above the target
	allLatenciesAboveTarget := true
	for _, res := range results {
		if res.Latency(latencyPercentile) <= float64(targetLatency) {
			allLatenciesAboveTarget = false
			break
		}
	}
	if allLatenciesAboveTarget {
		return fmt.Errorf("all observed latencies are above the target latency of %dms", targetLatency)
	}

	// If all checks pass, the prediction is valid
	return nil
}

func interpolateThroughput(results []*TestResult, predictedConnections float64) (float64, error) {
	// Ensure we have enough results for interpolation
	if len(results) < 2 {
		return 0, fmt.Errorf("not enough test results to interpolate")
	}

	// Find the test results adjacent to predictedConnections
	var predictedLow, predictedHigh *TestResult
	for _, res := range results {
		if float64(res.Connections) >= predictedConnections {
			predictedHigh = res
			break
		}
		predictedLow = res
	}

	// Check if adjacent results were found
	if predictedLow == nil || predictedHigh == nil {
		return 0, fmt.Errorf("predicted connections %.2f is out of bounds for the test results", predictedConnections)
	}

	// Perform linear interpolation
	x1, y1 := float64(predictedLow.Connections), predictedLow.Throughput
	x2, y2 := float64(predictedHigh.Connections), predictedHigh.Throughput
	predictedThroughput := y1 + (y2-y1)*(predictedConnections-x1)/(x2-x1)

	return predictedThroughput, nil
}

func analyzeAndPredict(targetLatency int, latencyPercentile LatencyPercentile, results []*TestResult) (float64, error) {
	// Perform quadratic regression
	a, b, c, err := quadraticRegression(results, latencyPercentile)
	if err != nil {
		return 0, fmt.Errorf("failed to perform quadratic regression: %w", err)
	}

	// Predict concurrency for 100ms latency using quadratic regression
	predictedConcurrencyQuad, err := predictConcurrencyQuad(a, b, c, targetLatency)
	if err != nil {
		return 0, fmt.Errorf("failed to predict concurrency: %w", err)
	}

	// Validate the prediction
	if err := validatePrediction(a, b, c, targetLatency, latencyPercentile, results); err != nil {
		return 0, err
	}

	predictedRPS, err := interpolateThroughput(results, predictedConcurrencyQuad)
	if err != nil {
		return 0, fmt.Errorf("failed to interpolate throughput: %w", err)
	}

	// Print analysis results
	fmt.Printf("\nAnalysis Results:\n")
	fmt.Printf("Quadratic regression equation: Latency (ms) = %.2fx^2 + %.2fx + %.2f\n", a, b, c)
	fmt.Printf("Predicted Concurrency for %dms Latency: %.2f\n", targetLatency, predictedConcurrencyQuad)
	fmt.Printf("Predicted RPS for %dms (Concurrency %.2f): %.2f\n", targetLatency, predictedConcurrencyQuad, predictedRPS)

	return predictedConcurrencyQuad, nil
}
