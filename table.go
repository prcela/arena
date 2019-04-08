package main

type Table struct {
	ID                string       `json:"id"`
	PlayersID         []string     `json:"players_id"`
	Bet               int64        `json:"bet"`
	Private           bool         `json:"private"`
	PlayersReady      []string     `json:"players_ready,omitempty"` // for league only
	PlayersForRematch []string     `json:"players_for_rematch,omitempty"`
}
