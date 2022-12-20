# Rock - Paper - Scissors Multigame

**Important: this project is in progress. This is not a final (production) software. It's a development test**

Based on: https://github.com/lukemaster/rock-paper-scissors and https://github.com/gobwas/ws

This software is composed by a Go server, and a reactjs client application.

These allows people to play this game in local network.

There is a master player who is able to configure the rounds of the game.

In adition, you can change your player nick name.

**Important: not tested in games with more than two players**

### If you don't have Go installed, please, install it before.
In MacOs:
>
> - brew install golang

### For launching it, first, you have to install all dependences:
>
> - cd rock-paper-scissors-golang
> - npm run install-complete

### Now, you can run it:

> - cd rock-paper-scissors-golang
> - npm start

### For developing, and debugging better, you can run this two services separately:

Open one terminal and type:
> - npm run server

Now, open another terminal, and type:

> - cd rock-paper-scissors-golang
> - npm run start-react

### For getting and running a production app, follow this steps:

> - cd rock-paper-scissors-golang
> - npm run build

Now, you can run it:

> - npm run start-built


Good Luck!