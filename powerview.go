// Copyright (c) 2016 The Go Authors. All rights reserved.
// See LICENSE.

// Package powerview speaks the Hunter Douglas PowerView Hub protocol
// to control motorized blinds & shades.
package powerview

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Hub struct {
	ip string
}

func (h *Hub) do(path string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", "http://"+h.ip+path, nil)
	if strings.HasSuffix(path, "?") {
		// Trailing question mark matters! Otherwise hub hangs.
		// See https://github.com/golang/go/issues/13488#issuecomment-172232677
		req.URL.Opaque = path
	}
	cancel := make(chan struct{})
	req.Cancel = cancel
	timer := time.AfterFunc(500*time.Millisecond, func() { close(cancel) })
	defer timer.Stop()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("powerview hub: %v", res.Status)
	}
	return res, nil
}

// NewHub returns a new Hub on the local LAN.
func NewHub(ip string) *Hub {
	return &Hub{ip: ip}
}

type Scenes struct {
	h   *Hub
	res scenesJSON
}

type Scene struct {
	Hub  *Hub
	ID   int64
	Name string
	Room *Room
}

func (s *Scene) Do() error {
	if s == nil {
		return errors.New("nil scene")
	}
	res, err := s.Hub.do("/api/scenes?sceneid=" + fmt.Sprint(s.ID))
	if err != nil {
		return err
	}
	res.Body.Close()
	return nil
}

type Room struct {
	h  *Hub
	ID int64
}

func (s *Scenes) Map() map[string]*Scene {
	m := map[string]*Scene{}
	for _, d := range s.res.Scenes {
		s := &Scene{
			Hub:  s.h,
			Name: string(d.Name),
			ID:   d.ID,
			Room: &Room{
				ID: d.RoomID,
				h:  s.h,
			},
		}
		m[s.Name] = s
	}
	return m
}

func (s *Scenes) ByName() []*Scene {
	m := s.Map()
	names := make([]string, 0, len(m))
	scenes := make([]*Scene, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		scenes = append(scenes, m[name])
	}
	return scenes
}

func (h *Hub) Scenes() (*Scenes, error) {
	res, err := h.do("/api/scenes?") // trailing question mark matters
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	sl := &Scenes{h: h}
	if err := json.NewDecoder(res.Body).Decode(&sl.res); err != nil {
		return nil, err
	}
	return sl, nil
}

type scenesJSON struct {
	SceneIDs []int64          `json:"sceneIds"`
	Scenes   []*sceneDataJSON `json:"sceneData"`
}

type sceneDataJSON struct {
	ID      int64  `json:"id"`
	Name    []byte `json:"name"`
	RoomID  int64  `json:"roomId"`
	Order   int64  `json:"order"`
	ColorID int64  `json:"colorId"`
	IconID  int64  `json:"iconId"`
}
