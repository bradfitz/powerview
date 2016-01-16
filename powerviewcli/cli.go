// Copyright (c) 2016 The Go Authors. All rights reserved.
// See LICENSE.

// Command powerviewcli interacts with a Hunter Douglas PowerView Hub
// to control motorized blinds & shades.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/bradfitz/powerview"
)

var ip = flag.String("ip", "10.0.0.31", "IP address of hub")

func main() {
	flag.Parse()
	h := powerview.NewHub(*ip)
	sl, err := h.Scenes()
	if err != nil {
		log.Fatal(err)
	}
	if a := flag.Args(); len(a) == 2 && a[0] == "scene" {
		if err := sl.Map()[a[1]].Do(); err != nil {
			log.Fatalf("error switching to scene %q: %v", a[1], err)
		}
		return
	}

	for _, s := range sl.ByName() {
		fmt.Printf("Scene %d: %v\n", s.ID, s.Name)
	}
}
