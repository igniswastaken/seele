package test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/zerothy/seele/service"
)

func TestRebalancing(t *testing.T) {
	fmt.Println("--- Testing Consistent Hashing ---")
	ring := service.NewHashRing(50)
	nodes := []string{"NodeA", "NodeB", "NodeC"}
	for _, node := range nodes {
		ring.AddNode(node)
	}

	const numKeys = 100000
	locationMap := make(map[string]string)

	for i := 0; i < numKeys; i++ {
		key := "Key-" + strconv.Itoa(i)
		locationMap[key] = ring.GetNode(key)
	}

	fmt.Println("Adding NodeD to the ring...")
	ring.AddNode("NodeD")

	moved := 0
	for i := 0; i < numKeys; i++ {
		key := "Key-" + strconv.Itoa(i)
		newNode := ring.GetNode(key)
		if locationMap[key] != newNode {
			moved++
		}
	}

	percentMoved := (float64(moved) / float64(numKeys)) * 100
	fmt.Printf("Consistent Hashing: Moved %d keys (%.2f%%)\n", moved, percentMoved)
	fmt.Println("Ideal is 25% (1/4 of data moves to new node)")
	fmt.Println("----------------------------------")
}
