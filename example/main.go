package main

import (
	"fmt"

	"github.com/hashrabbit/honeybadger"
)

var hb = honeybadger.New("changeme")

func main() {
	defer func() {
		if rec := recover(); rec != nil {
			id, err := hb.Report(rec)
			if err != nil {
				fmt.Printf("Failed to report error %s\n", err)
				panic(err)
			}
			fmt.Printf("Recovered and reported error %s\n", id)
		}
	}()

	_, err := hb.Reportf("a %s error: %v", "formatted", []string{"foo", "bar", "baz"})
	if err != nil {
		fmt.Printf("Failed to report message %s\n", err)
	}

	panic("oh no!")
}
