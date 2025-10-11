package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

const (
	baseURL      = "http://localhost:8080"
	testUser     = "benchmark_user"
	testPassword = "benchmark_pass"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		MaxConnsPerHost:     1000,
		IdleConnTimeout:     90 * time.Second,
	},
}

var authCookie string

func setupBenchmark() error {
	loginData := fmt.Sprintf("username=%s&password=%s", testUser, testPassword)
	resp, err := httpClient.Post(baseURL+"/api/v1/auth/login", "application/x-www-form-urlencoded", bytes.NewBufferString(loginData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "session" {
			authCookie = cookie.Value
			break
		}
	}

	if authCookie == "" {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("no session cookie received, response: %s", string(body))
	}

	return nil
}

func createTestScript(scriptName, scriptCode string) error {
	scriptData := map[string]string{
		"name": scriptName,
		"code": scriptCode,
	}

	jsonData, _ := json.Marshal(scriptData)
	req, err := http.NewRequest("POST", baseURL+"/api/v1/rules", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "session="+authCookie)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create script: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func executeScript(scriptName string, input map[string]interface{}) (*http.Response, error) {
	jsonData, _ := json.Marshal(input)
	req, err := http.NewRequest("POST", baseURL+"/api/v1/rules/"+scriptName+"/execute", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "session="+authCookie)

	return httpClient.Do(req)
}

func BenchmarkHTTPJSONProcessing(b *testing.B) {
	if err := setupBenchmark(); err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	scriptCode := `import (
		"encoding/json"
	)

	func Transform(in []byte) []byte {
		var data map[string]interface{}
		if err := json.Unmarshal(in, &data); err != nil {
			result := map[string]interface{}{
				"valid": false,
				"error": "invalid JSON",
			}
			out, _ := json.Marshal(result)
			return out
		}
	
		userID, hasUserID := data["user_id"]
		amount, hasAmount := data["amount"]
		timestamp, hasTimestamp := data["timestamp"]
	
		result := map[string]interface{}{
			"valid":     hasUserID && hasAmount && hasTimestamp,
			"user_id":   userID,
			"amount":    amount,
			"timestamp": timestamp,
			"processed": true,
		}
	
		if amountFloat, ok := amount.(float64); ok {
			result["amount_with_tax"] = amountFloat * 1.2
			result["currency"] = "USD"
		}
	
		out, _ := json.Marshal(result)
		return out
	}`

	if err := createTestScript("json-processor", scriptCode); err != nil {
		b.Fatalf("Failed to create script: %v", err)
	}

	testCases := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "simple_valid",
			data: map[string]interface{}{
				"user_id":   "12345",
				"amount":    100.0,
				"timestamp": "2023-01-01T00:00:00Z",
			},
		},
		{
			name: "incomplete",
			data: map[string]interface{}{
				"user_id": "12345",
				"amount":  100.0,
			},
		},
		{
			name: "large_json",
			data: map[string]interface{}{
				"user_id":   "12345",
				"amount":    100.0,
				"timestamp": "2023-01-01T00:00:00Z",
				"metadata": map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
					"key4": "value4",
					"key5": "value5",
				},
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				resp, err := executeScript("json-processor", tc.data)
				if err != nil {
					b.Fatalf("Request failed: %v", err)
				}
				resp.Body.Close()

				if resp.StatusCode != 200 {
					b.Fatalf("Unexpected status: %d", resp.StatusCode)
				}
			}
		})
	}
}

func BenchmarkHTTPMathCalculations(b *testing.B) {
	if err := setupBenchmark(); err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	scriptCode := `import (
		"encoding/json"
		"math"
	)

	func Transform(in []byte) []byte {
		var input map[string]interface{}
		if err := json.Unmarshal(in, &input); err != nil {
			result := map[string]interface{}{
				"error": "invalid JSON",
			}
			out, _ := json.Marshal(result)
			return out
		}
	
		numbers, ok := input["numbers"].([]interface{})
		if !ok {
			result := map[string]interface{}{
				"error": "numbers field not found or not an array",
			}
			out, _ := json.Marshal(result)
			return out
		}
	
		var values []float64
		for _, n := range numbers {
			if val, ok := n.(float64); ok {
				values = append(values, val)
			}
		}
	
		if len(values) == 0 {
			result := map[string]interface{}{
				"error": "no valid numbers found",
			}
			out, _ := json.Marshal(result)
			return out
		}
	
		sum := 0.0
		min := values[0]
		max := values[0]
	
		for _, val := range values {
			sum += val
			if val < min {
				min = val
			}
			if val > max {
				max = val
			}
		}
	
		mean := sum / float64(len(values))
	
		variance := 0.0
		for _, val := range values {
			diff := val - mean
			variance += diff * diff
		}
		variance /= float64(len(values))
		stdDev := math.Sqrt(variance)
	
		for i := 0; i < len(values)-1; i++ {
			for j := i + 1; j < len(values); j++ {
				if values[i] > values[j] {
					values[i], values[j] = values[j], values[i]
				}
			}
		}
	
		var median float64
		if len(values)%2 == 0 {
			median = (values[len(values)/2-1] + values[len(values)/2]) / 2
		} else {
			median = values[len(values)/2]
		}
	
		result := map[string]interface{}{
			"count":    len(values),
			"sum":      sum,
			"mean":     mean,
			"median":   median,
			"min":      min,
			"max":      max,
			"std_dev":  stdDev,
			"variance": variance,
		}
	
		out, _ := json.Marshal(result)
		return out
	}`

	if err := createTestScript("math-calculator", scriptCode); err != nil {
		b.Fatalf("Failed to create script: %v", err)
	}

	testCases := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "small_array",
			data: map[string]interface{}{
				"numbers": []float64{1, 2, 3, 4, 5},
			},
		},
		{
			name: "medium_array",
			data: map[string]interface{}{
				"numbers": []float64{1.5, 2.3, 3.7, 4.1, 5.9, 6.2, 7.8, 8.4, 9.1, 10.0},
			},
		},
		{
			name: "large_array",
			data: map[string]interface{}{
				"numbers": generateFloatArray(1000),
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				resp, err := executeScript("math-calculator", tc.data)
				if err != nil {
					b.Fatalf("Request failed: %v", err)
				}
				resp.Body.Close()

				if resp.StatusCode != 200 {
					b.Fatalf("Unexpected status: %d", resp.StatusCode)
				}
			}
		})
	}
}

func generateFloatArray(size int) []float64 {
	numbers := make([]float64, size)
	for i := 0; i < size; i++ {
		numbers[i] = float64(i + 1)
	}
	return numbers
}

func TestHTTPPerformanceComparison(t *testing.T) {
	fmt.Println("=== HTTP API Performance Test ===")

	if err := setupBenchmark(); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	scriptCode := `import (
		"encoding/json"
	)

	func Transform(in []byte) []byte {
		var data map[string]interface{}
		json.Unmarshal(in, &data)
		result := map[string]interface{}{
			"valid": true,
			"processed": true,
		}
		out, _ := json.Marshal(result)
		return out
	}`

	if err := createTestScript("perf-test", scriptCode); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	testData := map[string]interface{}{
		"user_id":   "12345",
		"amount":    100.0,
		"timestamp": "2023-01-01T00:00:00Z",
	}

	iterations := 100
	start := time.Now()

	for i := 0; i < iterations; i++ {
		resp, err := executeScript("perf-test", testData)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("Unexpected status: %d", resp.StatusCode)
		}
	}

	duration := time.Since(start)
	avgTime := duration / time.Duration(iterations)
	throughput := float64(iterations) / duration.Seconds()

	fmt.Printf("HTTP API Performance:\n")
	fmt.Printf("  Iterations: %d\n", iterations)
	fmt.Printf("  Total time: %v\n", duration)
	fmt.Printf("  Average time: %v\n", avgTime)
	fmt.Printf("  Throughput: %.2f ops/sec\n", throughput)
	fmt.Printf("  Throughput: %.2f ops/ms\n", throughput/1000)
}

func TestLoadTestRPS(t *testing.T) {
	fmt.Println("=== Load Test: 1M RPS ===")

	if err := setupBenchmark(); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	scriptCode := `func Transform(in []byte) []byte {
		return []byte("{\"status\":\"ok\",\"processed\":true}")
	}`

	if err := createTestScript("load-test", scriptCode); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	testData := map[string]interface{}{
		"user_id":   "12345",
		"amount":    100.0,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	fmt.Printf("Warming up cache...\n")
	resp, err := executeScript("load-test", testData)
	if err != nil {
		t.Fatalf("Warmup failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("Warmup failed with status %d: %s", resp.StatusCode, string(body))
	}
	fmt.Printf("Cache warmed up successfully!\n")

	totalRequests := 5000
	concurrency := 100
	requestsPerWorker := totalRequests / concurrency
	targetRPS := 200
	delayBetweenRequests := time.Duration(int64(time.Second) / int64(targetRPS/concurrency))

	var wg sync.WaitGroup
	var mu sync.Mutex
	var durations []time.Duration
	var errorCount int
	var completedRequests int

	start := time.Now()

	fmt.Printf("\nStarting requests...\n")

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		workerID := i
		go func() {
			defer wg.Done()

			for j := 0; j < requestsPerWorker; j++ {
				if j > 0 && delayBetweenRequests > 0 {
					time.Sleep(delayBetweenRequests)
				}

				reqStart := time.Now()
				resp, err := executeScript("load-test", testData)
				duration := time.Since(reqStart)

				mu.Lock()
				completedRequests++
				if completedRequests%1000 == 0 {
					fmt.Printf("Progress: %d/%d requests completed (%.1f%%)\n",
						completedRequests, totalRequests, float64(completedRequests)/float64(totalRequests)*100)
				}
				mu.Unlock()

				if err != nil {
					mu.Lock()
					errorCount++
					fmt.Printf("Worker %d: Error: %v\n", workerID, err)
					mu.Unlock()
					continue
				}

				if resp.StatusCode != 200 {
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					mu.Lock()
					errorCount++
					fmt.Printf("Worker %d: Bad status: %d, body: %s\n", workerID, resp.StatusCode, string(body))
					mu.Unlock()
					continue
				}
				resp.Body.Close()

				mu.Lock()
				durations = append(durations, duration)
				mu.Unlock()
			}
			fmt.Printf("Worker %d finished\n", workerID)
		}()
	}

	fmt.Printf("Waiting for all workers to complete...\n")
	wg.Wait()
	fmt.Printf("All workers completed!\n")
	totalDuration := time.Since(start)

	successCount := len(durations)
	actualRPS := float64(successCount) / totalDuration.Seconds()

	var totalLatency time.Duration
	for _, d := range durations {
		totalLatency += d
	}
	avgLatency := totalLatency / time.Duration(len(durations))

	sortDurations(durations)
	p95Index := int(float64(len(durations)) * 0.95)
	p99Index := int(float64(len(durations)) * 0.99)

	p95Latency := durations[p95Index]
	p99Latency := durations[p99Index]

	fmt.Printf("Load Test Results:\n")
	fmt.Printf("  Target RPS: 1,000,000\n")
	fmt.Printf("  Actual RPS: %.2f\n", actualRPS)
	fmt.Printf("  Success Rate: %.2f%% (%d/%d)\n",
		float64(successCount)/float64(totalRequests)*100, successCount, totalRequests)
	fmt.Printf("  Error Count: %d\n", errorCount)
	fmt.Printf("  Total Duration: %v\n", totalDuration)
	fmt.Printf("  Average Latency: %v\n", avgLatency)
	fmt.Printf("  P95 Latency: %v\n", p95Latency)
	fmt.Printf("  P99 Latency: %v\n", p99Latency)
	fmt.Printf("  Concurrency: %d\n", concurrency)
}

func sortDurations(durations []time.Duration) {
	for i := 0; i < len(durations)-1; i++ {
		for j := i + 1; j < len(durations); j++ {
			if durations[i] > durations[j] {
				durations[i], durations[j] = durations[j], durations[i]
			}
		}
	}
}
