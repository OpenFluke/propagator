package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	startPort  = 10002
	portStep   = 3
	numPods    = 10
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

	// Filter successful pods
	var successfulPods []PodResult
	for _, res := range results { // 'results' from previous operations
		if res.Success {
			successfulPods = append(successfulPods, res)
		}
	}

	//nuke(successfulPods)

	// Run the load balancing test
	//loadBalancingTest(successfulPods)
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

// loadBalancingTest performs a load balancing test by spawning cubes at planet coordinates.
func loadBalancingTest(pods []PodResult) {
	// Define the batch sizes for the test
	batchSizes := []int{10 /*20 40, 50*/ /*200, 300, 400, 500*/}

	// Iterate over each batch size
	for _, batchSize := range batchSizes {
		fmt.Printf("Starting load balancing test with %d cubes per planet\n", batchSize)
		// Remove all cubes before starting the batch
		//nuke(pods)
		var wg sync.WaitGroup

		// Process each pod concurrently
		for _, pod := range pods {
			wg.Add(1)
			go func(pod PodResult) {
				defer wg.Done()

				// Connect to the pod
				conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", pod.Host, pod.Port), time.Duration(timeoutSec)*time.Second)
				if err != nil {
					fmt.Printf("  Failed to connect to %s:%d: %v\n", pod.Host, pod.Port, err)
					return
				}
				defer conn.Close()

				// Authenticate with the pod
				if err := send(conn, authPass); err != nil {
					fmt.Printf("  Failed to send auth to %s:%d: %v\n", pod.Host, pod.Port, err)
					return
				}
				authResp := read(conn)
				if !strings.Contains(authResp, "auth_success") {
					fmt.Printf("  Authentication failed for %s:%d: %s\n", pod.Host, pod.Port, authResp)
					return
				}

				// Spawn cubes for each planet
				for _, planet := range pod.Planets {
					position := planet.Position
					x, xOk := position["x"]
					y, yOk := position["y"]
					z, zOk := position["z"]
					if !xOk || !yOk || !zOk {
						fmt.Printf("  Invalid position for planet %s on %s:%d\n", planet.Name, pod.Host, pod.Port)
						continue
					}

					// Spawn 'batchSize' cubes per planet
					for i := 0; i < batchSize; i++ {
						// Add a small random offset to avoid overlap
						offsetX := rand.Float64()*2 - 1 // Random float between -1 and 1
						offsetY := rand.Float64()*2 - 1
						offsetZ := rand.Float64()*2 - 1
						spawnPos := []float64{x + offsetX, y + offsetY, z + offsetZ}

						// Create the spawn_cube command
						spawnCmd := map[string]interface{}{
							"type":     "spawn_cube",
							"position": spawnPos,
							"rotation": []float64{0, 0, 0}, // Default rotation
						}
						cmdJSON, err := json.Marshal(spawnCmd)
						if err != nil {
							fmt.Printf("  Failed to marshal spawn command: %v\n", err)
							continue
						}

						// Send the spawn command
						if err := send(conn, string(cmdJSON)); err != nil {
							fmt.Printf("  Failed to send spawn command to %s:%d: %v\n", pod.Host, pod.Port, err)
							return
						}
					}
				}
				fmt.Printf("  Completed spawning %d cubes on %d planets for pod %s:%d\n", batchSize, len(pod.Planets), pod.Host, pod.Port)
			}(pod)
		}

		// Wait for all pods to complete spawning for this batch
		wg.Wait()
		fmt.Printf("Batch of %d cubes per planet finished. Waiting 10 seconds...\n", batchSize)
		time.Sleep(10 * time.Second)
	}

	// Remove all cubes before starting the batch
	//nuke(pods)
	fmt.Println("Load balancing test completed successfully.")
}

func nuke(pods []PodResult) {
	var wg sync.WaitGroup

	for _, pod := range pods {
		wg.Add(1)
		go func(pod PodResult) {
			defer wg.Done()

			addr := fmt.Sprintf("%s:%d", pod.Host, pod.Port)
			conn, err := net.DialTimeout("tcp", addr, time.Duration(timeoutSec)*time.Second)
			if err != nil {
				fmt.Printf("  Failed to connect to %s: %v\n", addr, err)
				return
			}
			defer conn.Close()

			if err := send(conn, authPass); err != nil {
				fmt.Printf("  Auth error on %s: %v\n", addr, err)
				return
			}
			if !strings.Contains(read(conn), "auth_success") {
				fmt.Printf("  Auth failed on %s\n", addr)
				return
			}

			// Max retries
			maxRetries := 5

			for attempt := 1; attempt <= maxRetries; attempt++ {
				if err := send(conn, `{"type":"get_cube_list"}`); err != nil {
					fmt.Printf("  Failed to get cube list on %s: %v\n", addr, err)
					return
				}
				raw := read(conn)

				var cubeData map[string]interface{}
				if err := json.Unmarshal([]byte(raw), &cubeData); err != nil {
					fmt.Printf("  JSON error from %s: %v\n", addr, err)
					return
				}

				cubes := toStringArray(cubeData["cubes"])
				if len(cubes) == 0 {
					fmt.Printf("  All cubes cleared on %s\n", addr)
					break
				}

				for _, cube := range cubes {
					cmd := fmt.Sprintf(`{"type":"despawn_cube","cube_name":"%s"}`, cube)
					if err := send(conn, cmd); err != nil {
						fmt.Printf("  Failed to despawn %s on %s: %v\n", cube, addr, err)
					}
				}

				fmt.Printf("  NUKED %d cubes on %s (pass %d)\n", len(cubes), addr, attempt)
				time.Sleep(500 * time.Millisecond) // Let the server process it
			}
		}(pod)
	}

	wg.Wait()
	fmt.Println("NUKE phase completed with verification passes.")
}
