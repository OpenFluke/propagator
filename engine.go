package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	startPort  = 10002
	portStep   = 3
	numPods    = 100
	authPass   = "my_secure_password"
	endMarker  = "<???DONE???---"
	timeoutSec = 10 // 10-second timeout
)

var hosts = []string{
	"192.168.0.229",
	"192.168.0.227",
}

// PodResult holds the result of a connection attempt to a pod.
type PodResult struct {
	Host    string
	Port    int
	Success bool
	Error   string
	Cubes   []string
	Planets []interface{}
}

func main() {
	// Record the start time for timing the entire operation
	startTime := time.Now()

	// Synchronization tools
	var wg sync.WaitGroup
	resultsChan := make(chan PodResult, numPods*len(hosts)) // Buffered channel to collect results

	// Launch goroutines to connect to all pods concurrently
	for _, host := range hosts {
		for i := 0; i < numPods; i++ {
			port := startPort + i*portStep
			wg.Add(1)
			go func(host string, port int) {
				defer wg.Done()
				result := checkPod(host, port)
				resultsChan <- result
			}(host, port)
		}
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(resultsChan) // Close the channel after all goroutines are done

	// Collect and process results
	var results []PodResult
	totalCubes := 0
	totalPlanets := 0
	successCount := 0

	for result := range resultsChan {
		results = append(results, result)
		if result.Success {
			successCount++
			totalCubes += len(result.Cubes)
			totalPlanets += len(result.Planets)
		}
	}

	// Print the summary
	fmt.Println("\n=== MULTIVERSE SUMMARY ===")
	for _, res := range results {
		if res.Success {
			fmt.Printf("[%s:%d] Connected successfully. Cubes: %d, Planets: %d\n",
				res.Host, res.Port, len(res.Cubes), len(res.Planets))
		} else {
			fmt.Printf("[%s:%d] Failed to connect: %s\n", res.Host, res.Port, res.Error)
		}
	}

	fmt.Printf("\nTotal successful connections: %d / %d\n", successCount, numPods*len(hosts))
	fmt.Printf("TOTAL CUBES ACROSS MULTIVERSE: %d\n", totalCubes)
	fmt.Printf("TOTAL PLANETS ACROSS MULTIVERSE: %d\n", totalPlanets)

	// Calculate and display the total time taken
	elapsed := time.Since(startTime)
	fmt.Printf("Total time taken: %s\n", elapsed)
}

// checkPod attempts to connect to a pod, authenticate, and retrieve data.
func checkPod(host string, port int) PodResult {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, time.Duration(timeoutSec)*time.Second)
	if err != nil {
		return PodResult{
			Host:    host,
			Port:    port,
			Success: false,
			Error:   fmt.Sprintf("Failed to connect: %v", err),
		}
	}
	defer conn.Close()

	// Send authentication
	send(conn, authPass)
	authResp := read(conn)
	if !strings.Contains(authResp, "auth_success") {
		return PodResult{
			Host:    host,
			Port:    port,
			Success: false,
			Error:   "Authentication failed or invalid response",
		}
	}

	// Get cubes
	send(conn, `{"type":"get_cube_list"}`)
	cubesRaw := read(conn)
	var cubeData map[string]interface{}
	if err := json.Unmarshal([]byte(cubesRaw), &cubeData); err != nil {
		return PodResult{
			Host:    host,
			Port:    port,
			Success: false,
			Error:   fmt.Sprintf("Failed to parse cube list: %v", err),
		}
	}
	cubes := toStringArray(cubeData["cubes"])

	// Get planets
	send(conn, `{"type":"get_planets"}`)
	planetsRaw := read(conn)
	var planetData map[string]interface{}
	if err := json.Unmarshal([]byte(planetsRaw), &planetData); err != nil {
		return PodResult{
			Host:    host,
			Port:    port,
			Success: false,
			Error:   fmt.Sprintf("Failed to parse planet list: %v", err),
		}
	}
	planets := toInterfaceArray(planetData["planets"])

	// Success case
	return PodResult{
		Host:    host,
		Port:    port,
		Success: true,
		Cubes:   cubes,
		Planets: planets,
	}
}

// send writes a message with the end marker.
func send(conn net.Conn, msg string) {
	_, err := conn.Write([]byte(msg + endMarker))
	if err != nil {
		fmt.Printf("  Error sending message to %s: %v\n", conn.RemoteAddr().String(), err)
	}
}

// read reads until the end marker is found.
func read(conn net.Conn) string {
	reader := bufio.NewReader(conn)
	conn.SetReadDeadline(time.Now().Add(timeoutSec * time.Second))
	var buf bytes.Buffer

	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err != io.EOF {
				fmt.Printf("  Read error from %s: %v\n", conn.RemoteAddr().String(), err)
			}
			break
		}
		buf.WriteByte(b)
		if buf.Len() >= len(endMarker) && strings.HasSuffix(buf.String(), endMarker) {
			break
		}
	}

	msg := buf.String()
	if len(msg) >= len(endMarker) {
		return msg[:len(msg)-len(endMarker)]
	}
	return msg
}

// toStringArray converts an interface{} to a string slice.
func toStringArray(v interface{}) []string {
	if v == nil {
		return []string{}
	}
	arrI := v.([]interface{})
	arr := make([]string, 0, len(arrI))
	for _, x := range arrI {
		arr = append(arr, fmt.Sprintf("%v", x))
	}
	return arr
}

// toInterfaceArray converts an interface{} to an interface slice.
func toInterfaceArray(v interface{}) []interface{} {
	if v == nil {
		return []interface{}{}
	}
	return v.([]interface{})
}
