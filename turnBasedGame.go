package main

type TurnBasedGame interface {
	enterPlayer(playerId string)
	exitPlayer(playerId string)
	createTable(playerId string, capacity int, private bool) *Table
}
