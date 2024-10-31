package main

import (
	"fmt"
	"time"
)

func sleepBackground() {
	go func() {
		fmt.Printf("Sleeping for 5 seconds\n")
		time.Sleep(5 * time.Second)
		fmt.Printf("Done sleeping\n")
	}()
}

func main() {
	fmt.Printf("Hello, world!\n")
	sleepBackground()

	for {
		time.Sleep(1 * time.Second)
		fmt.Printf("Still alive\n")
	}
}
