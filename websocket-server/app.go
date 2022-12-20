package main

import (
	//common
	"log"

	//websocket
	"flag"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
	"websocket_server_rock_paper_scissors/gopool"

	"github.com/gobwas/ws"
	"github.com/google/uuid"
	"github.com/mailru/easygo/netpoll"

	//rest API
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Player struct {
	Uuid         string
	io           sync.Mutex
	conn         io.ReadWriteCloser
	out          chan []byte
	pool         *gopool.Pool
	mu           sync.RWMutex
	MasterPlayer bool
	Choice       int8
	Score        int
	RoundScore   int
	Result       uint8
	Results      map[uint]Object //map[string]interface{}
	Nickname     string
	desc         *netpoll.Desc
}

type connectionInstance struct {
	Secret string
	Name   string
	Rounds int8
}

const SECRET string = "LET'S_PLAY"
const SOCKET_PORT string = "4001"

// Socket server/connection config
var (
	addr = flag.String("listen", ":"+SOCKET_PORT, "address to bind to")
	//debug     = flag.String("pprof", "", "address for pprof http")
	workers   = flag.Int("workers", 128, "max workers count")
	queue     = flag.Int("queue", 1, "workers task queue size")
	ioTimeout = flag.Duration("io_timeout", time.Millisecond*100, "i/o operations timeout")
)

// deadliner is a wrapper around net.Conn that sets read/write deadlines before
// every Read() or Write() call.
type deadliner struct {
	net.Conn
	t time.Duration
}

func nameConn(conn net.Conn) string {
	return conn.LocalAddr().String() + " > " + conn.RemoteAddr().String()
}

func manageNewConnection(c *gin.Context) {

	var newInstance connectionInstance

	if err := c.BindJSON(&newInstance); err == nil {
		if newInstance.Secret != SECRET {
			c.IndentedJSON(http.StatusForbidden, gin.H{"error": "wrong_secret"})
		} else {
			c.IndentedJSON(http.StatusAccepted, gin.H{"message": "right_secret", "socket_port": SOCKET_PORT})
		}
	}
	c.IndentedJSON(http.StatusNoContent, gin.H{"error": "wrong_secret"})
}

func main() {

	var (
		//exit = make(chan struct{})   ----> NOT NEEDED (more information at bottom)
		//pollerAcceptMu sync.Mutex
		resumerWaiter sync.WaitGroup
	)

	// //REST API router configuration ///
	router := gin.Default()
	router.Use(cors.Default())
	//
	// METHOD FOR GETTING WEBSOCKET'S PORT
	router.POST("/connection", manageNewConnection)
	//////////////////////////////////////

	poller, err := netpoll.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	// Create incoming connections listener for websocket
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("websocket is listening on %s", ln.Addr().String())

	var pool = gopool.NewPool(*workers, *queue, 1)

	game := initGame(pool, &poller)

	// accept is a channel to signal about next incoming connection Accept()
	// results.
	accept := make(chan error, 1) //ACCEPTED INCOMING CONNECTION

	// handle is a new incoming connection handler.
	// It upgrades TCP connection to WebSocket, registers netpoll listener on
	// it and stores it as a game player in Game instance.
	//
	// We will call it below within accept() loop.
	handle := func(conn net.Conn) { //NEW INCOMING CONNECTION HANDLER
		// NOTE: we wrap conn here to show that ws could work with any kind of
		// io.ReadWriter.
		safeConn := deadliner{conn, *ioTimeout}

		// Zero-copy upgrade to WebSocket connection.
		hs, err := ws.Upgrade(safeConn)
		if err != nil {
			log.Printf("%s: upgrade error: %v", nameConn(conn), err)
			conn.Close()
			return
		}

		log.Printf("%s: established websocket connection: %+v", nameConn(conn), hs)

		var numberOfPlayers = len(game.players)

		if numberOfPlayers >= 2 { // RESTRICTION: 2 PLAYERS
			log.Print("accept error: number of players exceded")
			//defer func() { accept <- nil }()
			conn.Close()
			return
		}

		// Register incoming player in game.
		player := game.Register(uuid.New().String(), safeConn, pool)
		if player == nil {
			conn.Close()
			return
		}
		// Create netpoll event descriptor for conn.
		// We want to handle only read events of it.
		desc := netpoll.Must(netpoll.HandleRead(conn))

		player.desc = desc

		// Subscribe to events about conn.
		poller.Start(desc, func(ev netpoll.Event) {
			if ev&(netpoll.EventReadHup|netpoll.EventHup) != 0 {
				// When ReadHup or Hup received, this mean that client has
				// closed at least write end of the connection or connections
				// itself. So we want to stop receive events about such conn
				// and remove it from the game registry.
				poller.Stop(desc)
				game.Remove(player)
				return
			}

			// Here we can read some new message from connection.
			// We can not read it right here in callback, because then we will
			// block the poller's inner loop.
			// We do not want to spawn a new goroutine to read single message.
			// But we want to reuse previously spawned goroutine.
			pool.Schedule(func() {
				if err := game.Receive(player); err != nil {
					// When receive failed, we can only disconnect broken
					// connection and stop to receive events about it.
					poller.Stop(desc)
					game.Remove(player)
				}
			})
		})
		player.emit("your-uuid", Object{"uuid": player.Uuid})
	}

	// Create netpoll descriptor for the listener.
	// We use OneShot here to manually resume events stream when we want to.
	acceptDesc := netpoll.Must( //HELPER FOR PREVENTING ERROR
		netpoll.HandleListener( //DESCRIPTOR FOR LISTENER ln, AND AVAILABLES EVENTS FOR THIS LISTENER
			ln,
			netpoll.EventRead|netpoll.EventOneShot,
		))

	resumer := func() {
		//executing Resume since other context prevents deadlock inside poller.Start
		resumerWaiter.Wait()
		poller.Resume(acceptDesc)
	}

	poller.Start(acceptDesc, func(e netpoll.Event) {

		resumerWaiter.Add(1)

		go resumer()

		err := pool.ScheduleTimeout(time.Millisecond, func() { //EXECUTE THIS FUNCTION EVERY time.Millisecond
			conn, err := ln.Accept()
			if err != nil {
				accept <- err
				return
			}

			accept <- nil
			handle(conn) //REAL FUNCTION TO EXECUTE
		})
		if err == nil {
			err = <-accept
		}
		if err != nil {
			/*ne*/ _, ok := err.(net.Error)
			if err != gopool.ErrScheduleTimeout || ok /* && ne.Temporary() */ { //ne.Temporary() DEPRECATED
				delay := 5 * time.Millisecond
				log.Printf("accept error: %v; retrying in %s", err, delay)
				time.Sleep(delay)
			}
			log.Fatalf("accept error: %v", err)
		}

		resumerWaiter.Done()

	})

	router.Run(":8080")

	//NOT NEEDED BECAUSE router.Run keeps app running
	//<-exit
}
