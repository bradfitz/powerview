// Copyright (c) 2016 The Go Authors. All rights reserved.
// See LICENSE.

// Package powerview speaks the Hunter Douglas PowerView Hub protocol
// to control Hunter Douglas PowerView motorized blinds & shades.
//
// This package is barely functional. It could be much more polished.
package powerview

// Notes:
// $ curl http://10.0.0.31/api/userdata/
// {"userData":{"serialNumber":"xxxxx","rfID":"0xB1C2","rfIDInt":45506,"rfStatus":0,"hubName":"TWFzdGVy","macAddress":"00:26:74:xx:xx:xx","roomCount":1,"shadeCount":6,"groupCount":1,"sceneCount":5,"sceneMemberCount":30,"multiSceneCount":0,"multiSceneMemberCount":0,"scheduledEventCount":0,"sceneControllerCount":0,"sceneControllerMemberCount":0,"localTimeDataSet":true,"enableScheduledEvents":true,"remoteConnectEnabled":true,"editingEnabled":true,"_isEnableFirmwareDownload":true,"_isEnableRemoteActionServer":false,"unassignedShadeCount":0,"undefinedShadeCount":0}}

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Hub represents a PowerView Hub.
type Hub struct {
	ip string
}

func (h *Hub) get(path string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", "http://"+h.ip+path, nil)
	if strings.HasSuffix(path, "?") {
		// Trailing question mark matters! Otherwise hub hangs.
		// See https://github.com/golang/go/issues/13488#issuecomment-172232677
		req.URL.Opaque = path
	}
	return h.do(req)
}

func (h *Hub) do(req *http.Request) (*http.Response, error) {
	cancel := make(chan struct{})
	req.Cancel = cancel
	timer := time.AfterFunc(1*time.Second, func() { close(cancel) })
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

// Scenes is a list of scenes.
type Scenes struct {
	h   *Hub
	res scenesJSON
}

// Scene is a scene previously configured on the PowerView hub.
type Scene struct {
	Hub  *Hub
	ID   int64
	Name string
	Room *Room
}

// Do activates the scene s.
func (s *Scene) Do() error {
	if s == nil {
		return errors.New("nil scene")
	}
	res, err := s.Hub.get("/api/scenes?sceneid=" + fmt.Sprint(s.ID))
	if err != nil {
		return err
	}
	res.Body.Close()
	return nil
}

// Room is a pre-configured room.
type Room struct {
	h    *Hub
	ID   int64
	Name string // empty if fetched via Scenes
}

// Map returns the scenes keyed by name.
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

// ByName returns a slice of scenes in the list s, sorted by
// their name.
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

// Map returns the shades keyed by name.
func (s *Shades) Map() map[string]*Shade {
	m := map[string]*Shade{}
	for _, d := range s.res.Shades {
		s := &Shade{
			Hub:             s.h,
			Name:            string(d.Name),
			ID:              d.ID,
			BatteryStrength: d.BatteryStrength,
			BatteryStatus:   d.BatteryStatus,
			BatteryIsLow:    d.BatteryIsLow,
			Bottom:          d.Positions["position1"],
			Top:             d.Positions["position2"],
		}
		m[s.Name] = s
	}
	return m
}

// ByName returns a slice of shades in the list s, sorted by
// their name.
func (s *Shades) ByName() []*Shade {
	m := s.Map()
	names := make([]string, 0, len(m))
	shades := make([]*Shade, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		shades = append(shades, m[name])
	}
	return shades
}

// Scenes queries the hub to find a list of scenes.
func (h *Hub) Scenes() (*Scenes, error) {
	res, err := h.get("/api/scenes?") // trailing question mark matters
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

type Rooms struct {
	h   *Hub
	res roomsJSON
}

// Map returns the rooms keyed by name.
func (s *Rooms) Map() map[string]*Room {
	m := map[string]*Room{}
	for _, d := range s.res.Rooms {
		s := &Room{
			h:    s.h,
			Name: string(d.Name),
			ID:   d.ID,
		}
		m[s.Name] = s
	}
	return m
}

// ByName returns a slice of rooms in the list s, sorted by
// their name.
func (s *Rooms) ByName() []*Room {
	m := s.Map()
	names := make([]string, 0, len(m))
	rooms := make([]*Room, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		rooms = append(rooms, m[name])
	}
	return rooms
}

// Rooms queries the hub to find a list of rooms.
func (h *Hub) Rooms() (*Rooms, error) {
	res, err := h.get("/api/rooms?") // trailing question mark matters
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	rs := &Rooms{h: h}
	if err := json.NewDecoder(res.Body).Decode(&rs.res); err != nil {
		return nil, err
	}
	return rs, nil
}

// Shades is a list of shades.
type Shades struct {
	h   *Hub
	res shadesJSON
}

// Shade is an individual shade.
type Shade struct {
	Hub             *Hub
	ID              int64
	Name            string
	BatteryStrength int64
	BatteryStatus   int64
	BatteryIsLow    bool

	// Bottom (internally "position1") is the position of the
	// shade's bottom. 65535 is the top. 0 is the bottom.
	Bottom uint16

	// Top (internally "position2") is the position of the shade's
	// top.  0 is the top. 65535 is the bottom.
	Top uint16
}

// Move moves the shade.
func (s *Shade) Move(bottom, top uint16) error {
	req, err := http.NewRequest("PUT", "http://"+s.Hub.ip+"/api/shades/"+fmt.Sprint(s.ID),
		strings.NewReader(fmt.Sprintf(`{"shade":{"id":%d,"positions":{"posKind2":2,"position2":%d,"posKind1":1,"position1":%d}}}`,
			s.ID, top, bottom)))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	if err != nil {
		return err
	}
	res, err := s.Hub.do(req)
	if err != nil {
		return err
	}
	s.Bottom = bottom
	s.Top = top
	// TODO: parse response? kinda boring:
	//12:02:07.742226 IP 10.0.0.31.80 > 10.0.0.247.58587: Flags [P.], seq 159:386, ack 334, win 1127, length 227
	//0x0000:  4500 010b 0043 0000 4006 6495 0a00 001f  E....C..@.d.....
	//0x0010:  0a00 00f7 0050 e4db 6f25 29e0 cb89 1880  .....P..o%).....
	//0x0020:  5018 0467 36cd 0000 7b22 7368 6164 6522  P..g6...{"shade"
	//0x0030:  3a7b 2269 6422 3a36 3535 3133 2c22 6e61  :{"id":65513,"na
	//0x0040:  6d65 223a 2254 5746 7a64 4756 794e 673d  me":"TWFzdGVyNg=
	//0x0050:  3d22 2c22 726f 6f6d 4964 223a 3430 3339  =","roomId":4039
	//0x0060:  362c 2267 726f 7570 4964 223a 3435 3930  6,"groupId":4590
	//0x0070:  362c 226f 7264 6572 223a 322c 2274 7970  6,"order":2,"typ
	//0x0080:  6522 3a38 2c22 6261 7474 6572 7953 7472  e":8,"batteryStr
	//0x0090:  656e 6774 6822 3a36 312c 2262 6174 7465  ength":61,"batte
	//0x00a0:  7279 5374 6174 7573 223a 312c 2262 6174  ryStatus":1,"bat
	//0x00b0:  7465 7279 4973 4c6f 7722 3a66 616c 7365  teryIsLow":false
	//0x00c0:  2c20 2270 6f73 6974 696f 6e73 223a 7b22  ,."positions":{"
	//0x00d0:  706f 7369 7469 6f6e 3122 3a30 2c22 706f  position1":0,"po
	//0x00e0:  734b 696e 6431 223a 312c 2270 6f73 6974  sKind1":1,"posit
	//0x00f0:  696f 6e32 223a 3538 3335 312c 2270 6f73  ion2":58351,"pos
	//0x0100:  4b69 6e64 3222 3a32 7d7d 7d              Kind2":2}}}
	res.Body.Close()
	return nil
}

// Shades queries the hub to find a list of shades.
func (h *Hub) Shades() (*Shades, error) {
	res, err := h.get("/api/shades?") // trailing question mark matters
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	sl := &Shades{h: h}
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

type roomsJSON struct {
	RoomIDs []int64         `json:"roomIds"`
	Rooms   []*roomDataJSON `json:"roomData"`
}

type roomDataJSON struct {
	ID      int64  `json:"id"`
	Name    []byte `json:"name"`
	Order   int64  `json:"order"`
	ColorID int64  `json:"colorId"`
	IconID  int64  `json:"iconId"`
}

type shadesJSON struct {
	ShadeIDs []int64          `json:"shadeIds"`
	Shades   []*shadeDataJSON `json:"shadeData"`
}

type shadeDataJSON struct {
	ID              int64             `json:"id"`
	Name            []byte            `json:"name"`
	GroupID         int64             `json:"groupId"`
	Order           int64             `json:"order"`
	Type            int64             `json:"type"`
	BatteryStrength int64             `json:"batteryStrength"`
	BatteryStatus   int64             `json:"batteryStatus"`
	BatteryIsLow    bool              `json:"batteryIsLow"`
	Positions       map[string]uint16 `json:"positions"`
}
