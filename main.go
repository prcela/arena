package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
)

func loadConfiguration() Config {
	var file string
	if runtime.GOOS == "darwin" {
		file = "configDev.json"
	} else {
		file = "/home/prcela/work/src/github.com/prcela/kolo/configProd.json"
	}
	var config Config
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		fmt.Println(err.Error())
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config
}

func main() {

	log.Println("Arena")

	// s1 := rand.NewSource(time.Now().UnixNano())
	// r1 := rand.New(s1)
	// msgCounter = r1.Int31()

	config := loadConfiguration()

	arena := newArena()
	go arena.hub.run(config)

	http.HandleFunc("/chat", func(w http.ResponseWriter, req *http.Request) {
		log.Println("request", req)
		arena.hub.ServeWs(w, req)
	})

	http.HandleFunc("/info", func(w http.ResponseWriter, req *http.Request) {

		db, s := GetDatabaseSessionCopy()
		defer s.Close()

		dbPlayersCt, _ := db.C("players").Count()

		info := struct {
			MinRequiredVersion int `json:"min_required_version"`
			RoomMainCt         int `json:"room_main_ct"`
			DBPlayersCount     int `json:"db_players_ct"`
		}{
			MinRequiredVersion: config.MinRequiredVersion, // postavi na 60 kad se nakupi dovoljno igrača na toj verziji
			RoomMainCt:         0,                         // TODO
			DBPlayersCount:     dbPlayersCt,
		}

		js, err := json.Marshal(info)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)

	})

	http.ListenAndServe(config.Addr, nil)

}
