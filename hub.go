package main

import (
	"encoding/json"
	"github.com/fatih/color"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"strconv"
	"time"
)

// hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	arena   *Arena
	clients map[*Client]bool

	chRegisterClient   chan *Client
	chUnregisterClient chan *Client
	chWaitClient       chan *Client

	chAction             chan *Action
	chBroadcastAll       chan []byte
	chBroadcast          chan Broadcast
	chProcessDiff        chan bool
	chVerifyBroadcastNum chan int32

	broadcastsToAck map[int32]*Broadcast
	differ          *RoomDiffer
}

func newHub(arena *Arena) *Hub {
	return &Hub{
		arena:                arena,
		clients:              make(map[*Client]bool),
		chRegisterClient:     make(chan *Client),
		chUnregisterClient:   make(chan *Client),
		chWaitClient:         make(chan *Client),
		chAction:             make(chan *Action),
		chBroadcastAll:       make(chan []byte, 256),
		chBroadcast:          make(chan Broadcast, 256),
		chProcessDiff:        make(chan bool, 256),
		chVerifyBroadcastNum: make(chan int32),
		broadcastsToAck:      make(map[int32]*Broadcast),
		differ:               newRoomDiffer(),
	}
}

func (hub *Hub) run(config Config) {
	log.Println("Hub run")

	for {
		select {
		case client := <-hub.chRegisterClient:
			hub.clients[client] = true
			hub.onClientRegistered(client)

		case client := <-hub.chUnregisterClient:
			log.Println("hub.unregisterClient")
			if _, ok := hub.clients[client]; ok {
				// remove player if it is not in game
				if player := hub.arena.players[client.playerID]; player != nil {
					// TODO: ...
					log.Printf("unregister client: player: %v", player.Alias)
				}
				delete(hub.clients, client)
				close(client.send)
			}

		case client := <-hub.chWaitClient:
			if player := hub.arena.players[client.playerID]; player != nil {
				go func() {
					if player.waitingForClient {
						player.chWaitClient <- client
					}
					if client.wasConnected {
						hub.resendMissedMessages(player.ID, player.missedMessages)
					} else {
						hub.arena.excludePlayerFromAllGames(player)
					}
				}()
				player.missedMessages = []MissedMessage{}
			}

		case action := <-hub.chAction:

			switch action.name {
			case "arena_info":
				info := hub.arena.info()
				js, _ := json.Marshal(info)
				hub.chBroadcast <- Broadcast{
					playersID: []string{action.fromPlayerID},
					message:   js,
				}
			}

		case broadcast := <-hub.chBroadcast:
			if broadcast.msgNum != 0 {
				hub.broadcastsToAck[broadcast.msgNum] = &broadcast
				go func(msgNum int32) {
					time.Sleep(5 * time.Second)
					hub.chVerifyBroadcastNum <- msgNum
				}(broadcast.msgNum)
			}
			for _, playerID := range broadcast.playersID {
				if player := hub.arena.players[playerID]; player != nil {
					foundClient := false
					for client := range hub.clients {
						if playerID == client.playerID {
							foundClient = true
							select {
							case client.send <- broadcast.message:
							default:
								if broadcast.msgNum != 0 {
									color.Yellow("append missed message")
									player.missedMessages = append(player.missedMessages, MissedMessage{message: broadcast.message, msgNum: broadcast.msgNum})
								}
								go func(c *Client) {
									hub.chUnregisterClient <- client
								}(client)
							}
						}
					}
					if broadcast.msgNum != 0 {
						if foundClient {
							player.toAck[broadcast.msgNum] = true
							color.Yellow("player: %v, to ack: %v", player.Alias, player.toAck)
						} else {
							log.Println("Client not found")
							color.Yellow("append missed message")
							player.missedMessages = append(player.missedMessages, MissedMessage{message: broadcast.message, msgNum: broadcast.msgNum})
						}
					}
				}
			}

		case message := <-hub.chBroadcastAll:
			for client := range hub.clients {
				select {
				case client.send <- message:
				default:
					go func(c *Client) {
						hub.chUnregisterClient <- client
					}(client)
				}
			}

		case msgNum := <-hub.chVerifyBroadcastNum:
			if b := hub.broadcastsToAck[msgNum]; b != nil {
				// try broadcast again to left players
				color.HiGreen("Resending broadcast %v", msgNum)
				hub.chBroadcast <- *b
			} else {
				color.Green("Verify broadcast %v ✓ Done", msgNum)
			}

		case <-hub.chProcessDiff:
			log.Println("Ušo u chProcessDiff")
			js := hub.differ.diff(hub)
			hub.chBroadcastAll <- js

		}
	}
}

func (hub *Hub) onClientRegistered(c *Client) {
	db, s := GetDatabaseSessionCopy()
	defer s.Close()

	player := hub.arena.players[c.playerID]

	foundInArena := player != nil

	if foundInArena {
		// player already exists and we were waiting for this client
		log.Println("found player in arena", player.ID)
		log.Println("wait channel: ", player.chWaitClient)

		go func(c *Client) {
			color.Cyan("hub.waitClient <- c")
			hub.chWaitClient <- c
		}(c)
	} else {
		log.Println("Not found player in arena")
		err := db.C("players").FindId(c.playerID).One(&player)
		if err != nil && c.deviceUUID != nil {
			log.Printf("Searching for playerId: %v by device uuid: %v\n", c.playerID, *c.deviceUUID)
			// try to find player by deviceUUID
			err = db.C("players").Find(bson.M{"device_uuid": *c.deviceUUID}).One(&player)
		}
		if player != nil {
			c.playerID = player.ID
			player.missedMessages = []MissedMessage{}
			player.chWaitClient = make(chan *Client)
			player.toAck = make(map[int32]bool)
			player.changed = true
		} else {
			log.Printf("Not found player in database\n")
			log.Println(err)
			log.Println("New player:", c.playerID)
			player = newPlayer()
			if len(c.playerID) > 0 {
				player.ID = c.playerID
				player.DeviceUUID = c.deviceUUID
			} else {
				player.ID = bson.NewObjectId().Hex()
				player.DeviceUUID = c.deviceUUID
				player.Alias = "Player " + player.ID[len(player.ID)-4:]
				c.playerID = player.ID
				log.Println("Created new player ID: ", player.ID)
			}
			db.C("players").Insert(player)
		}

		hub.arena.players[player.ID] = player
	}

	js, err := json.Marshal(struct {
		MsgFunc      string  `json:"msg_func"`
		Player       *Player `json:"player"`
		FoundInArena bool    `json:"found_in_arena"`
	}{
		MsgFunc:      "player_stat",
		Player:       player,
		FoundInArena: foundInArena,
	})

	if err != nil {
		log.Println(err)
	}

	hub.chBroadcast <- Broadcast{
		playersID: []string{player.ID},
		message:   js,
	}

	if !foundInArena {
		hub.chProcessDiff <- true
	}
}

func (hub *Hub) resendMissedMessages(playerID string, missedMessages []MissedMessage) {
	color.Blue("resendMissedMessages: %v", len(missedMessages))

	for _, mm := range missedMessages {

		log.Println("Player: ", playerID)
		log.Println("Player hub: ", hub)
		hub.chBroadcast <- Broadcast{playersID: []string{playerID}, message: mm.message, msgNum: mm.msgNum}

		var dic struct {
			Turn string `json:"turn"`
		}
		// ako je turn message
		if err := json.Unmarshal(mm.message, &dic); err == nil {
			if dic.Turn == "rd" {
				// roll dice
				time.Sleep(1100 * time.Millisecond)
			} else {
				time.Sleep(200 * time.Millisecond)
			}
		}
	}
}

// ServeWs handles websocket requests from the peer.
func (hub *Hub) ServeWs(w http.ResponseWriter, r *http.Request) {
	log.Println("ServeWs")
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	var cookiePlayerId, _ = r.Cookie("playerId")
	var cookieWasConnected, _ = r.Cookie("wasConnected")
	var cookieDeviceUUID, _ = r.Cookie("deviceUUID")
	var cookieVersion, _ = r.Cookie("version")
	var cookieGameId, _ = r.Cookie("gameId")

	wasConnected := cookieWasConnected != nil && cookieWasConnected.Value == "true"
	var deviceUUID *string
	if cookieDeviceUUID != nil {
		deviceUUID = &cookieDeviceUUID.Value
	}

	var version = 0
	if cookieVersion != nil {
		version, _ = strconv.Atoi(cookieVersion.Value)
	}

	var gameId *string
	if cookieGameId != nil {
		gameId = &cookieGameId.Value
	}

	client := &Client{
		hub:          hub,
		conn:         conn,
		playerID:     cookiePlayerId.Value,
		gameId:       gameId,
		wasConnected: wasConnected,
		version:      version,
		deviceUUID:   deviceUUID,
		send:         make(chan []byte, 256),
	}
	log.Println("Created client")

	hub.chRegisterClient <- client
	log.Println("Registered client")

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
