package main

// Player ...
type Player struct {
	ID         string  `json:"id" bson:"_id,omitempty"`
	DeviceUUID *string `json:"device_uuid,omitempty" bson:"device_uuid,omitempty"`
	Alias      string  `json:"alias" bson:"alias"`
	Flag       string  `json:"flag" bson:"flag"`
	Diamonds   int64   `json:"diamonds" bson:"diamonds"`
	Pretzels   int64   `json:"pretzels" bson:"pretzels"`
	Retentions []int   `json:"retentions" bson:"retentions"`

	Achieved  []int    `json:"achieved,omitempty" bson:"achieved,omitempty"`
	CtInvited int      `json:"ct_invited,omitempty" bson:"ct_invited,omitempty"`
	Friends   []string `json:"friends,omitempty" bson:"friends,omitempty"`

	FcmToken *string `json:"fcm_token,omitempty" bson:"fcm_token,omitempty"`

	missedMessages []MissedMessage
	toAck          map[int32]bool
	changed        bool

	chWaitClient     chan *Client // wait for new client
	waitingForClient bool
}

func newPlayer() *Player {
	return &Player{
		Alias:            "New player",
		Flag:             "🏳️",
		Diamonds:         100,
		Pretzels:         0,
		Retentions:       []int{},
		missedMessages:   []MissedMessage{},
		toAck:            make(map[int32]bool),
		chWaitClient:     make(chan *Client),
		waitingForClient: false,
		changed:          true,
	}
}
