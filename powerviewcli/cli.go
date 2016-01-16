// Copyright (c) 2016 The Go Authors. All rights reserved.
// See LICENSE.

// Command powerviewcli interacts with a Hunter Douglas PowerView Hub
// to control motorized blinds & shades.
package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"

	"github.com/bradfitz/powerview"
)

var ip = flag.String("ip", "10.0.0.31", "IP address of hub")

func main() {
	flag.Parse()
	h := powerview.NewHub(*ip)
	sl, err := h.Scenes()
	if err != nil {
		log.Fatalf("Error getting scenes: %v", err)
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

	rooms, err := h.Rooms()
	if err != nil {
		log.Fatalf("Error getting rooms: %v", err)
	}
	for _, r := range rooms.ByName() {
		fmt.Printf("Room %d: %v\n", r.ID, r.Name)
	}

	shades, err := h.Shades()
	if err != nil {
		log.Fatalf("Error getting shades: %v", err)
	}
	for _, s := range shades.ByName() {
		fmt.Printf("Shade %d: %v %+v\n", s.ID, s.Name, s)
	}

	if a := flag.Args(); len(a) == 4 && a[0] == "shade" {
		name := a[1]
		s := shades.Map()[name]
		if s == nil {
			log.Fatalf("Unknown shade %q", s)
		}
		pos1, _ := strconv.Atoi(a[2])
		pos2, _ := strconv.Atoi(a[3])
		if err := s.Move(uint16(pos1), uint16(pos2)); err != nil {
			log.Fatalf("Error moving shade: %v", err)
		}
		log.Printf("Moved shade.")
	}

}
