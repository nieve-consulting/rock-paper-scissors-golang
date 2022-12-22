package main

import (
	"fmt"
	"log"
	"math"
	"net"
	"reflect"
	"strconv"
	"websocket_server_rock_paper_scissors/gopool"

	"github.com/mailru/easygo/netpoll"
)

/** ****** GAME STATES ******** */
var gameStates = []string{
	/// // complementary ////
	"ready",
	"attemping_to_game_setup",
	/// /////////////////////
	"playing",
	"restarting-game",
}

/** *************************** */

var rounds uint = 1

type Game struct {
	//mu sync.RWMutex --> MOVED TO PLAYER
	//numberOfPlayers uint
	players                    map[string]*Player
	playersRemaining           int8
	rounds                     int
	currentState               string
	restartGamePlayersNotified uint
	results                    map[int]Result

	pool   *gopool.Pool
	poller *netpoll.Poller
}

type Result struct {
	A    string
	B    string
	win  string
	lose string
}

func initGame(pool *gopool.Pool, poller *netpoll.Poller) *Game {
	game := &Game{
		pool:                       pool,
		players:                    make(map[string]*Player),
		playersRemaining:           0,
		rounds:                     1,
		currentState:               "ready",
		restartGamePlayersNotified: 0,
		results:                    make(map[int]Result),
		poller:                     poller,
	}

	return game
}

func (g *Game) socketDisconnect(player *Player) {
	(*(g.poller)).Stop(player.connDescriptor)
	g.Remove(player)
}

/** ****** ELEMENTS' LOGIC ****** */

// PAPER:    0 - WIN -> 2 ROCK
// SCISSORS: 1 - WIN -> 0 PAPER
// ROCK:     2 - WIN -> 1 SCISSORS
var elementsWin = []int8{
	2,
	0,
	1,
}

func checkResult(a int8, b int8) uint8 {
	if a != b {
		if !(elementsWin[a] != b) {
			return 1
		}
		return 0
	}
	return 2
}

func (g *Game) broadCastFatalError() {
	for k, player := range g.players {
		player.emit("fatal-error", nil)
		g.socketDisconnect(player)
		delete(g.players, k)
	}
	g.playersRemaining = 0
	rounds = 1
	for key := range g.results {
		delete(g.results, key)
	}

	g.currentState = gameStates[0]
}

func (g *Game) setPlayersRemaining(val int8) {
	g.playersRemaining = int8(math.Max(0, math.Min(float64(len(g.players)), float64(val))))
}

func (g *Game) gameResultN() {
	g.rounds = g.rounds - 1
	g.currentState = "playing"
	keys := reflect.ValueOf(g.players).MapKeys()

	g.results = make(map[int]Result)
	roundScores := make(map[string]int)
	for i := 0; i < len(keys); i++ {
		key := keys[i].Interface()
		var playerA = g.players[string(fmt.Sprintf("%v", key))]

		for j := i + 1; j < len(keys); j++ {
			key := keys[j].Interface()
			var playerB = g.players[string(fmt.Sprintf("%v", key))]

			var result = checkResult(playerA.Choice, playerB.Choice)
			/** ******** RESULT OPTIONS *********
			 * 0: LOSE
			 * 1: WIN
			 * 2: DRAW
			 ********** RESULT OPTIONS ******** */
			var win string
			var lose string

			if result != 2 {
				if result != 0 {
					win = playerA.Uuid
					lose = playerB.Uuid
				} else {
					win = playerB.Uuid
					lose = playerA.Uuid
				}
			}

			var res = Result{
				A:    playerA.Uuid,
				B:    playerB.Uuid,
				win:  win,
				lose: lose,
			}

			g.results[len(g.results)] = res
		}
		roundScores[playerA.Uuid] = 0
	}

	for _, result := range g.results {
		if len(result.win) > 0 {
			roundScores[result.win] += 1
			roundScores[result.lose] -= 1
		}
	}

	for _, result := range g.results {
		if len(result.win) > 0 {
			var playerWon = g.players[result.win]
			playerWon.Result = 1
			playerWon.RoundScore = roundScores[playerWon.Uuid]
			playerWon.Score += playerWon.RoundScore

			var playerLost = g.players[result.lose]
			playerLost.Result = 0
			playerLost.RoundScore = roundScores[playerLost.Uuid]
			playerLost.Score += playerLost.RoundScore

			playerWon.emit("result", Object{
				"newArray": []int8{playerWon.Choice, playerLost.Choice},
				"rounds":   g.rounds,
				"score":    playerWon.Score,
				"result":   playerWon.Result,
			})
			playerLost.emit("result", Object{
				"newArray": []int8{playerLost.Choice, playerWon.Choice},
				"rounds":   g.rounds,
				"score":    playerLost.Score,
				"result":   playerLost.Result,
			})
		} else {
			for _, element := range g.players {
				if element != nil {
					element.emit("result", Object{
						"newArray": []int8{element.Choice, element.Choice},
						"rounds":   g.rounds,
						"score":    element.Score,
						"result":   2,
					})
					element.RoundScore = 0
				} else {
					g.broadCastFatalError()
					return
				}
			}
		}
	}

	for playerKey, player := range g.players {
		var counter = len(player.Results)
		var choices = make(map[string]int8)
		var roundResults = make(map[string]bool)
		var scores = make(map[string]int)

		for key, pKey := range g.players {
			var nickname = pKey.Nickname
			if key == playerKey {
				nickname = "me"
			}

			var n bool = pKey.RoundScore > 0

			choices[nickname] = pKey.Choice
			roundResults[nickname] = n
			scores[nickname] = pKey.RoundScore
		}
		var obj Object = Object{
			"choices":      choices,
			"roundResults": roundResults,
			"scores":       scores,
		}
		player.Results[uint(counter)] = obj
	}

	if rounds <= 0 {
		rounds = 1
	}
}

func (g *Game) broadCastNewGame() {
	g.restartGamePlayersNotified = 0
	g.currentState = "restarting-game"

	for _, player := range g.players {
		player := player // For closure.
		g.pool.Schedule(func() {
			player.emit("restart-game", Object{})
			g.socketDisconnect(player)
		})
	}
	g.currentState = "attemping_to_game_setup"
}

func (g *Game) reConfigureGame(player *Player) {
	if g.currentState != "restarting-game" {
		/** ******* SETTING UP GAME ******* */
		if g.currentState == "attemping_to_game_setup" {
			player.emit("configure-game", Object{})
		}
		var newPlayersRemaining int8 = 0
		for _, element := range g.players {
			if element.Choice < 0 {
				newPlayersRemaining += 1
			}
		}
		g.setPlayersRemaining(newPlayersRemaining)
	}
}

func (g *Game) gameSetup(p *Player) {
	if len(g.players) > 1 {
		p.emit("let-us-play", Object{})
	}
}

func (g *Game) yourUuidAck(p *Player) {

	var numberOfPlayers = len(g.players)
	var masterPlayer bool = numberOfPlayers < 1

	g.setPlayersRemaining(g.playersRemaining + 1)
	if !masterPlayer {
		// be secure there is at least one master player
		var masterPlayerFound bool = false
		for _, element := range g.players {
			masterPlayerFound = masterPlayerFound || element.MasterPlayer
		}
		if masterPlayerFound {
			p.emit("join-game", nil)
		} else {
			pKeys := reflect.ValueOf(g.players).MapKeys()
			firstPKey := pKeys[0].Interface()
			var p = g.players[string(fmt.Sprintf("%v", firstPKey))]

			g.currentState = "attemping_to_game_setup"
			p.MasterPlayer = true
			g.reConfigureGame(p)
		}
	} else {
		g.currentState = "attemping_to_game_setup"
		p.MasterPlayer = true
		g.reConfigureGame(p)
		/** ******************************* */
	}
}

/** ******** CHOICE RECEPTION ********* */

func (g *Game) socketOnChoice(data Object, player *Player) {

	if data["choice"] != "" {
		var i int64
		i, err := strconv.ParseInt(fmt.Sprintf("%v", data["choice"]), 10, 8)
		if err != nil {
			g.broadCastFatalError()
		}
		player.Choice = int8(i)
		g.setPlayersRemaining(g.playersRemaining - 1)
		if g.playersRemaining <= 0 {
			if g.playersRemaining < 0 {
				g.broadCastFatalError()
			} else { // playersRemaining == 0
				g.gameResultN()
				g.playersRemaining = int8(len(g.players))
			}
		}
	} else {
		g.broadCastFatalError()
	}
}

func (g *Game) socketOnPlayerInfo(data Object, player *Player) {
	player.Choice = -1
	player.Score = 0
	player.Results = make(map[uint]Object)
	if player.MasterPlayer {
		i, err := strconv.ParseInt(fmt.Sprintf("%v", data["rounds"]), 10, 8)
		if err != nil {
			g.broadCastFatalError()
		}
		g.rounds = int(i)
	}
	player.Nickname = fmt.Sprintf("%v", data["nickname"])
}

func (g *Game) socketOnGetFinalScore(player *Player) {
	g.currentState = "attemping_to_game_setup"
	player.emit("show-final-score", Object{
		"finalScore": player.Results,
	})
}

func (g *Game) socketOnRestartGame() {
	g.restartGamePlayersNotified += 1
	if uint(len(g.players)) <= g.restartGamePlayersNotified {
		g.broadCastNewGame()
	}
}

// Receive and reads next message from player.
// It blocks until full message received.
func (g *Game) Receive(p *Player) error {

	req, err := p.readRequest()

	if err != nil {
		if err.Error() == "ws closed: 1001 " {
			//PREVENT REFRESHED OR CLOSED BROWSER,
			//PLAYER UNREGISTERING
			g.socketDisconnect(p)
			return nil
		} else {
			log.Fatal(err)
			p.conn.Close()
			return err
		}
	}
	if req == nil {
		// Handled some control message.
		return nil
	}

	switch req.Method {
	case "your-uuid-ACK":
		g.yourUuidAck(p)
	case "choice":
		g.socketOnChoice(req.Params, p)
	case "player-info":
		g.socketOnPlayerInfo(req.Params, p)
	case "get-final-score":
		g.socketOnGetFinalScore(p)
	case "restart-game":
		g.socketOnRestartGame()
	case "game-setup":
		g.gameSetup(p)
	default:
		return p.writeErrorTo(req, "Received message not implemented")
	}
	return nil
}

// Register registers new connection as a Player.
func (g *Game) Register(uuid string, conn net.Conn, pool *gopool.Pool) *Player {

	player := &Player{
		Uuid:         uuid,
		Choice:       -1,
		Score:        0,
		conn:         conn,
		MasterPlayer: false,
		out:          make(chan []byte, 1),
		pool:         pool,
		Results:      make(map[uint]Object),
	}

	//Player:out dispatcher over new thread
	go player.writer()
	///////////////////////////////////////

	g.players[uuid] = player

	return player
}

// Remove removes player from game.
func (g *Game) Remove(player *Player) {
	player.mu.Lock()
	removed := g.remove(player)
	player.mu.Unlock()

	if !removed {
		return
	}

	g.setPlayersRemaining(g.playersRemaining - 1)

	var keys = make([]string, 0, len(g.players))

	if player.MasterPlayer {
		if len(g.players) > 1 {
			var p = g.players[keys[1]]
			p.MasterPlayer = true
			p.Choice = -1
		}

		player.mu.Lock()
		removed := g.remove(player)
		player.mu.Unlock()

		if !removed {
			return
		}
		newKeys := reflect.ValueOf(g.players).MapKeys()
		if len(newKeys) > 0 {
			firstPKey := newKeys[0].Interface()
			var p = g.players[string(fmt.Sprintf("%v", firstPKey))]
			g.reConfigureGame(p)
		}
	} else {
		player.mu.Lock()
		removed := g.remove(player)
		player.mu.Unlock()

		if !removed {
			return
		}
	}
	fmt.Println(`Player disconnected. Players: ${Object.keys(players).length}`)
}

// mutex must be held.
func (g *Game) remove(player *Player) bool {
	if _, has := g.players[player.Uuid]; !has {
		return false
	}

	delete(g.players, player.Uuid)

	return true
}
