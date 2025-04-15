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

// Planet represents a planet object from the server.
type Planet struct {
	Position          map[string]float64   `json:"Position"`
	Seed              int                  `json:"Seed"`
	Name              string               `json:"Name"`
	ResourceLocations []map[string]float64 `json:"ResourceLocations"`
	TreeLocations     []map[string]float64 `json:"TreeLocations"`
	BiomeType         int                  `json:"BiomeType"`
}

// PodResult holds the result of a connection attempt to a pod.
type PodResult struct {
	Host    string
	Port    int
	Success bool
	Error   string
	Cubes   []string
	Planets []Planet
}

func main() {
	startTime := time.Now()
	var wg sync.WaitGroup
	resultsChan := make(chan PodResult, numPods*len(hosts))

	// Launch goroutines to check all pods concurrently
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

	wg.Wait()
	close(resultsChan)

	// Process results
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

	// Print summary
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
	fmt.Printf("Total time taken: %s\n", time.Since(startTime))
}

// checkPod connects to a pod, authenticates, and retrieves cube and planet data.
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

	// Authenticate
	if err := send(conn, authPass); err != nil {
		return PodResult{
			Host:    host,
			Port:    port,
			Success: false,
			Error:   fmt.Sprintf("Failed to send auth: %v", err),
		}
	}
	authResp := read(conn)
	if !strings.Contains(authResp, "auth_success") {
		return PodResult{
			Host:    host,
			Port:    port,
			Success: false,
			Error:   fmt.Sprintf("Authentication failed: %s", authResp),
		}
	}

	// Get cubes
	if err := send(conn, `{"type":"get_cube_list"}`); err != nil {
		return PodResult{
			Host:    host,
			Port:    port,
			Success: false,
			Error:   fmt.Sprintf("Failed to request cubes: %v", err),
		}
	}
	cubesRaw := read(conn)
	fmt.Printf("[%s:%d] Raw cube response: %s\n", host, port, cubesRaw)
	var cubeData map[string]interface{}
	if err := json.Unmarshal([]byte(cubesRaw), &cubeData); err != nil {
		return PodResult{
			Host:    host,
			Port:    port,
			Success: false,
			Error:   fmt.Sprintf("Failed to parse cube list: %v", err),
		}
	}
	if cubeData["type"] != "cube_list" {
		return PodResult{
			Host:    host,
			Port:    port,
			Success: false,
			Error:   fmt.Sprintf("Invalid cube response type: %v", cubeData["type"]),
		}
	}
	cubes := toStringArray(cubeData["cubes"])

	// Get planets
	if err := send(conn, `{"type":"get_planets"}`); err != nil {
		return PodResult{
			Host:    host,
			Port:    port,
			Success: false,
			Error:   fmt.Sprintf("Failed to request planets: %v", err),
		}
	}
	planetsRaw := read(conn)
	//fmt.Printf("[%s:%d] Raw planet response: %s\n", host, port, planetsRaw)
	var planetData map[string][]Planet
	if err := json.Unmarshal([]byte(planetsRaw), &planetData); err != nil {
		return PodResult{
			Host:    host,
			Port:    port,
			Success: false,
			Error:   fmt.Sprintf("Failed to parse planet list: %v", err),
		}
	}
	// Flatten the planets
	var allPlanets []Planet
	for _, planets := range planetData {
		allPlanets = append(allPlanets, planets...)
	}

	return PodResult{
		Host:    host,
		Port:    port,
		Success: true,
		Cubes:   cubes,
		Planets: allPlanets,
	}
}

// send writes a message with the end marker and returns any error.
func send(conn net.Conn, msg string) error {
	_, err := conn.Write([]byte(msg + endMarker))
	if err != nil {
		fmt.Printf("Error sending to %s: %v\n", conn.RemoteAddr().String(), err)
	}
	return err
}

// read reads until the end marker is found, with a timeout.
func read(conn net.Conn) string {
	reader := bufio.NewReader(conn)
	conn.SetReadDeadline(time.Now().Add(timeoutSec * time.Second))
	var buf bytes.Buffer
	chunk := make([]byte, 1024)
	for {
		n, err := reader.Read(chunk)
		if err != nil && err != io.EOF {
			fmt.Printf("Read error from %s: %v\n", conn.RemoteAddr().String(), err)
			break
		}
		buf.Write(chunk[:n])
		if strings.HasSuffix(buf.String(), endMarker) {
			break
		}
		if err == io.EOF {
			break
		}
	}
	msg := buf.String()
	if len(msg) >= len(endMarker) && strings.HasSuffix(msg, endMarker) {
		return msg[:len(msg)-len(endMarker)]
	}
	return msg
}

// toStringArray converts an interface{} to a string slice.
func toStringArray(v interface{}) []string {
	if v == nil {
		return []string{}
	}
	arrI, ok := v.([]interface{})
	if !ok {
		fmt.Printf("Warning: 'cubes' is not an array: %v\n", v)
		return []string{}
	}
	arr := make([]string, 0, len(arrI))
	for _, x := range arrI {
		arr = append(arr, fmt.Sprintf("%v", x))
	}
	return arr
}
