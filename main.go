package main

import (
	"fmt"

	config "github.com/AaroLankinen/blog_aggregator/internal/config"
)

func main() {
	cfg, err := config.ReadConfig()
	if err != nil {
		panic(err)
	}
	cfg.SetUser("aarol")
	fmt.Printf("Config: %+v\n", cfg)
}
