package main

import (
	"io"
	"sync"
	"websocket_server_rock_paper_scissors/gopool"

	"github.com/mailru/easygo/netpoll"
)

/******** CONN. AND COMM. OBJECTS ********/
// Object represents generic message parameters.
// In production environment it would be better to avoid such types for getting more performance.
type Object map[string]interface{}

type Request struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
	Params Object `json:"params"`
}

type Response struct {
	ID     int    `json:"id"`
	Result Object `json:"result"`
}

type Error struct {
	ID    int    `json:"id"`
	Error string `json:"error"`
}

/*****************************************/

/********** PLAYER ABSTRACTION ***********/
type Player struct {
	Uuid           string
	io             sync.Mutex
	conn           io.ReadWriteCloser
	out            chan []byte
	pool           *gopool.Pool
	mu             sync.RWMutex
	MasterPlayer   bool
	Choice         int8
	Score          int
	RoundScore     int
	Result         uint8
	Results        map[uint]Object
	Nickname       string
	connDescriptor *netpoll.Desc
}

/*****************************************/

/************** GAME STRUCT **************/
type Game struct {
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

/*****************************************/

/********** GAME ROUND RESULT ************/
type Result struct {
	A    string
	B    string
	win  string
	lose string
}

/*****************************************/

/*
******** SETTNG UP GAME STRUCT ***********

	Struct used for dealing websocket connection.
	Righ secret provided allows player to play the game.
	This message is used by REST API
*/
type connectionInstance struct {
	Secret string
	Name   string
	Rounds int8
}

/*****************************************/
