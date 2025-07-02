package main

import (
	"fmt"

	"github.com/ozaitsev92/gonewsbot/internal/config"
)

func main() {
	fmt.Println(config.Get())
}
