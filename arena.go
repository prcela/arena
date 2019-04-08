package main

import (
	"github.com/fatih/color"
)

type Arena struct {
	players map[string]*Player
	games   map[string]TurnBasedGame

	hub *Hub
}

func newArena() *Arena {
	arena := &Arena{
		players: make(map[string]*Player),
		games:   make(map[string]TurnBasedGame),
	}
	arena.hub = newHub(arena)
	return arena
}

func (arena *Arena) excludePlayerFromAllGames(player *Player) {
	color.Yellow("Exclude player %v from all games", player.Alias)
	// TODO...
}

func (arena *Arena) info() interface{} {
	i := struct {
		MsgFunc string                   `json:"msg_func"`
		Players map[string]*Player       `json:"players"`
		Games   map[string]TurnBasedGame `json:"games"`
	}{
		MsgFunc: "arena_info",
		Players: arena.players,
		Games:   arena.games,
	}
	return i
}
