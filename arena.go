package main

type Arena struct {
	hub *Hub
}

func newArena() *Arena {
	return &Arena{
		hub: newHub(),
	}
}
