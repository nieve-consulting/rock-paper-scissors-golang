# Rock - Paper - Scissors Multigame

**Important: this project is in progress. This is not a final (production) software. It's a development test**

Based on: https://github.com/lukemaster/rock-paper-scissors

This software is composed by a Go server, and a reactjs client application.

These allows people to play this game in local network.

There is a master player who is able to configure the rounds of the game.

In adition, you can change your player nick name.

**Important: not tested in games with more than two players**

### For launching it, first, you have to install all dependences:
>
> - cd server-socket-io
> - npm install

### Now, you can run it:

> - cd websocket-client
> - npm run rps

### For developing, and debugging better, you can run this two services separately:

Open one terminal and type:
> - cd websocket-server
> - go run *.go

Now, open another terminal, and type:

> - cd websocket-client
> - npm start


Good Luck!