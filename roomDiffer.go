package main

import (
	"encoding/json"
	"log"
)

type RoomDiffer struct {
	removedPlayersID []string
	players          map[string]*Player
}

func newRoomDiffer() *RoomDiffer {
	return &RoomDiffer{}
}

func (differ *RoomDiffer) diff(hub *Hub) []byte {
	log.Println("ušo u diff")
	// removed objects
	differ.players = make(map[string]*Player)

	for _, pID := range differ.removedPlayersID {
		differ.players[pID] = nil
	}

	differ.removedPlayersID = []string{}

	// collect changed objects
	for pID, p := range hub.players {
		if p != nil && p.changed {
			differ.players[pID] = p
			p.changed = false
		}
	}

	js, err := json.Marshal(struct {
		MsgFunc string             `json:"msg_func"`
		Players map[string]*Player `json:"players"`
	}{
		MsgFunc: "room_diff",
		Players: differ.players,
	})
	if err != nil {
		log.Println(err)
	}
	return js
}
