package main

type Config struct {
	Addr               string `json:"addr"`
	MinRequiredVersion int    `json:"min_required_version"`
	FsPath             string `json:"fs_path"`
}

type MissedMessage struct {
	message []byte
	msgNum  int32
}

// Broadcast message to desired players
type Broadcast struct {
	playersID []string
	message   []byte
	msgNum    int32
}


type Action struct {
	name         string
	message      []byte
	fromPlayerID string
	msgNum       int32
}