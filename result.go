package loadtest

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
)

type LatencyPercentile string

const (
	Latency50  LatencyPercentile = "50%"
	Latency90  LatencyPercentile = "90%"
	Latency98  LatencyPercentile = "98%"
	Latency99  LatencyPercentile = "99%"
	LatencyAvg LatencyPercentile = "avg"
)

// name,throughput,avg. latency,threads,connections,duration,completed,successful,errors,sockets,min. latency,max. latency,50%,90%,98%,99%
type TestResult struct {
	Name        string
	Throughput  float64
	AvgLatency  float64
	Threads     int
	Connections int
	Duration    float64
	Completed   int
	Successful  int
	Errors      int
	Sockets     int
	MinLatency  float64
	MaxLatency  float64
	Latency50   float64
	Latency90   float64
	Latency98   float64
	Latency99   float64
}

func (r *TestResult) String() string {
	sb := strings.Builder{}
	if r.Name != "" {
		sb.WriteString(fmt.Sprintf("Test: %s\n", r.Name))
	}
	sb.WriteString(fmt.Sprintf("Concurrency: %d\n", r.Connections))
	sb.WriteString(fmt.Sprintf("Throughput: %.2f RPS\n", r.Throughput))
	sb.WriteString(fmt.Sprintf("Avg. Latency: %.2fms\n", r.AvgLatency))
	sb.WriteString(fmt.Sprintf("Min Latency: %.2fms\n", r.MinLatency))
	sb.WriteString(fmt.Sprintf("Max Latency: %.2fms\n", r.MaxLatency))
	sb.WriteString(fmt.Sprintf("50%% Latency: %.2fms\n", r.Latency50))
	sb.WriteString(fmt.Sprintf("90%% Latency: %.2fms\n", r.Latency90))
	sb.WriteString(fmt.Sprintf("98%% Latency: %.2fms\n", r.Latency98))
	sb.WriteString(fmt.Sprintf("99%% Latency: %.2fms\n", r.Latency99))
	sb.WriteString(fmt.Sprintf("Threads: %d\n", r.Threads))
	sb.WriteString(fmt.Sprintf("Duration: %.2fs\n", r.Duration))
	sb.WriteString(fmt.Sprintf("Completed: %d\n", r.Completed))
	sb.WriteString(fmt.Sprintf("Successful: %d\n", r.Successful))
	sb.WriteString(fmt.Sprintf("Errors: %d\n", r.Errors))
	sb.WriteString(fmt.Sprintf("Sockets: %d\n", r.Sockets))
	return sb.String()
}

func (r *TestResult) Print(latencyPercentile LatencyPercentile) string {
	return fmt.Sprintf("Concurrency: %d, Throughput: %.2f RPS, Latency: %.2fms (%s)", r.Connections, r.Throughput, r.Latency(latencyPercentile), latencyPercentile)
}

func (r *TestResult) Latency(percentile LatencyPercentile) float64 {
	switch percentile {
	case Latency50:
		return r.Latency50
	case Latency90:
		return r.Latency90
	case Latency98:
		return r.Latency98
	case Latency99:
		return r.Latency99
	case LatencyAvg:
		return r.AvgLatency
	default:
		return -1
	}
}

func parseCSVOutput(output string) (*TestResult, error) {
	reader := csv.NewReader(strings.NewReader(output))

	// name,throughput,avg. latency,threads,connections,duration,completed,successful,errors,sockets,min. latency,max. latency,50%,90%,98%,99%
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println(output)
		return nil, fmt.Errorf("could not read CSV records: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("no records found in CSV output")
	}

	if len(records[0]) < 16 {
		return nil, fmt.Errorf("invalid CSV format \"%v\"", records[0])
	}

	res := &TestResult{}
	res.Name = records[0][0]
	throughput, err := strconv.ParseFloat(records[0][1], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse throughput: %w", err)
	}
	res.Throughput = throughput

	avgLatency, err := strconv.ParseFloat(records[0][2], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse avg latency: %w", err)
	}
	res.AvgLatency = avgLatency

	threads, err := strconv.Atoi(records[0][3])
	if err != nil {
		return nil, fmt.Errorf("failed to parse threads: %w", err)
	}
	res.Threads = threads

	connections, err := strconv.Atoi(records[0][4])
	if err != nil {
		return nil, fmt.Errorf("failed to parse connections: %w", err)
	}
	res.Connections = connections

	duration, err := strconv.ParseFloat(records[0][5], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse duration: %w", err)
	}
	res.Duration = duration

	completed, err := strconv.Atoi(records[0][6])
	if err != nil {
		return nil, fmt.Errorf("failed to parse completed: %w", err)
	}
	res.Completed = completed

	successful, err := strconv.Atoi(records[0][7])
	if err != nil {
		return nil, fmt.Errorf("failed to parse successful: %w", err)
	}
	res.Successful = successful

	errors, err := strconv.Atoi(records[0][8])
	if err != nil {
		return nil, fmt.Errorf("failed to parse errors: %w", err)
	}
	res.Errors = errors

	sockets, err := strconv.Atoi(records[0][9])
	if err != nil {
		return nil, fmt.Errorf("failed to parse sockets: %w", err)
	}
	res.Sockets = sockets

	minLatency, err := strconv.ParseFloat(records[0][10], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse min latency: %w", err)
	}
	res.MinLatency = minLatency

	maxLatency, err := strconv.ParseFloat(records[0][11], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse max latency: %w", err)
	}
	res.MaxLatency = maxLatency

	latency50, err := strconv.ParseFloat(records[0][12], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse 50%% latency: %w", err)
	}
	res.Latency50 = latency50

	latency90, err := strconv.ParseFloat(records[0][13], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse 90%% latency: %w", err)
	}
	res.Latency90 = latency90

	latency98, err := strconv.ParseFloat(records[0][14], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse 98%% latency: %w", err)
	}
	res.Latency98 = latency98

	latency99, err := strconv.ParseFloat(records[0][15], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse 99%% latency: %w", err)
	}
	res.Latency99 = latency99

	return res, nil
}
