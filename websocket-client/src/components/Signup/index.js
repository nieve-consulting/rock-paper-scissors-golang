import {Component} from 'react'

import RpsPage from '../RpsPage'

import './index.css'
import {
  PlayButton,
  PlayImage,
  HighLightName,
  HighLightTitle,
  ScoreImage,
  TableCentered,
  SignupContainer,
} from './styledComponents'

let ws = null

class Signup extends Component {
  choicesList = [
    {
      id: 0, // 'PAPER'
      imageUrl: '/assets/paper-image.png',
    },
    {
      id: 1, // 'SCISSORS'
      imageUrl: '/assets/scissor-image.png',
    },
    {
      id: 2, // 'ROCK'
      imageUrl: '/assets/rock-image.png',
    },
  ]

  uuid = ''

  constructor() {
    super()
    this.state = {
      nickname: 'player',
      rounds: 1,
      letsGame: false,
      gameState: 'waiting-game',
      score: 0,
      attempingResults: false,
      showResults: false,
      result: null,
      finalScore: false,
      results: [],
      newArray: [],
    }
  }

  getSocket = null

  componentDidMount() {
    fetch('http://localhost:8080/connection', {
      method: 'POST',
      responseType: 'json',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        Secret: "LET'S_PLAY",
        Name: 'Yoyo',
        Rounds: 2,
      }),
    })
      .then(res => res.json())
      .then(ret => {
        console.log(ret)
        const url = `ws://${window.location.hostname}:${ret.socket_port}`
        const p = new Promise((resolve, reject) => {
          ws = new WebSocket(url)
          let pending = true
          ws.onerror = err => {
            if (pending) {
              pending = false
              reject(err)
              return
            }

            console.warn(`websocket lifetime error:${err}`)
            Object.keys(pending).forEach(k => {
              pending[k].reject(err)
              delete pending[k]
            })
          }
          ws.onopen = () => {
            if (pending) {
              pending = false
              resolve(ws)
            }
          }
          ws.onmessage = s => {
            let msg
            try {
              msg = JSON.parse(s.data)
            } catch (err) {
              console.warn(`parse incoming message error:${err}`)
              return
            }

            // Notice
            if (msg.id === 0) {
              if (this.handleMethod[msg.method]) {
                let params = {}
                if (msg.params) {
                  params = msg.params
                }
                this.handleMethod[msg.method](params)
                return
              }
              console.warn(`no handler for method: ${msg.method}`)
              return
            }

            const request = pending[msg.id]
            if (request == null) {
              console.warn(`no pending request for: ${msg.method}, ${msg.id}`)
              return
            }

            delete pending[msg.id]
            if (msg.error != null) {
              request.reject(msg.error)
            } else {
              request.resolve(msg.result)
            }
          }
        })
        /* const url = `ws://${window.location.hostname}:${ret.socket_port}`
        const socket = socketIOClient(url)

        this.getSocket = () => socket

        socket.on('fatal-error', () => {
          this.setState({gameState: 'fatal-error'})
        })

        socket.on('your-uuid', data => {
          this.uuid = data?.uuid
          socket.emit('your-uuid-ACK')
        })

        socket.on('configure-game', () => {
          const {gameState} = this.state
          if (gameState !== 'waiting-game') {
            this.resetPlayer()
          }
          this.setState({gameState: 'configure-game'})
        })
        socket.on('join-game', () => {
          this.setState({gameState: 'join-game'})
        })
        socket.on('result', data => {
          if (data.rounds <= 0) {
            this.setState({finalScore: true})
          }
          this.setState({
            newArray: data.newArray,
            rounds: data.rounds,
            result: data.result,
            showResults: true,
            attempingResults: false,
            score: data.score,
          })
        })
        socket.on('show-final-score', data => {
          const finalScores = {}
          data.finalScore.forEach(round => {
            Object.keys(round.scores).forEach(key => {
              if (finalScores[key] === undefined) {
                finalScores[key] = 0
              }
              finalScores[key] += round.scores[key]
            })
          })
          const myData = data
          myData.finalScores = finalScores
          this.setState({results: myData})
          this.setState({letsGame: false, gameState: 'show-final-score'})
        })
        socket.on('restart-game', () => {
          window.location.reload()
        }) */
      })
      .catch(err => {
        console.log(err)
      })
  }

  handleFatalError = () => {
    this.setState({gameState: 'fatal-error'})
  }

  handleYourUuid = data => {
    console.log('bla')
    this.uuid = data?.uuid
    this.emit('your-uuid-ACK', {params: this.uuid})
  }

  handleConfigureGame = () => {
    const {gameState} = this.state
    if (gameState !== 'waiting-game') {
      this.resetPlayer()
    }
    this.setState({gameState: 'configure-game'})
  }

  handleJoinGame = () => {
    this.setState({gameState: 'join-game'})
  }

  handleResult = data => {
    if (data.rounds <= 0) {
      this.setState({finalScore: true})
    }
    this.setState({
      newArray: data.newArray,
      rounds: data.rounds,
      result: data.result,
      showResults: true,
      attempingResults: false,
      score: data.score,
    })
  }

  handleShowFinalScore = data => {
    const finalScores = {}
    Object.keys(data.finalScore).forEach(k => {
      const round = data.finalScore[k]
      Object.keys(round.scores).forEach(key => {
        if (finalScores[key] === undefined) {
          finalScores[key] = 0
        }
        finalScores[key] += round.scores[key]
      })
    })
    const myData = data
    myData.finalScores = finalScores
    this.setState({results: myData})
    this.setState({letsGame: false, gameState: 'show-final-score'})
  }

  handleRestartGame = () => {
    window.location.reload()
  }

  handleMethod = {
    'your-uuid': this.handleYourUuid,
    'join-game': this.handleJoinGame,
    'configure-game': this.handleConfigureGame,
    'show-final-score': this.handleShowFinalScore,
    result: this.handleResult,
    'restart-game': this.handleRestartGame,
  }

  emit = (method, parameters) => {
    ws.send(
      JSON.stringify({
        Method: method,
        params: parameters,
      }),
    )
  }

  play = () => {
    const {rounds, gameState, nickname} = this.state
    if (rounds <= 0) {
      return
    }
    const playerInfo = {}
    playerInfo.nickname = nickname
    if (gameState === 'configure-game') {
      playerInfo.rounds = parseInt(rounds)
    }
    this.emit('player-info', playerInfo)
    this.setState({letsGame: true})
  }

  roundChanges = e => {
    this.setState({rounds: e.target.value})
  }

  nicknameChanges = e => {
    this.setState({nickname: e.target.value})
  }

  sendChoice = choice => {
    console.log(this.uuid)
    this.emit('choice', {
      playerId: this.uuid,
      choice,
    })
    this.setState({attempingResults: true})
  }

  playAgain = () => {
    this.setState({showResults: false})
  }

  showFinalScore = () => {
    this.emit('get-final-score')
  }

  restartGame = () => {
    this.setState({gameState: 'waiting-for-restarting-game'})
    this.emit('restart-game')
  }

  resetPlayer = () => {
    this.setState({
      nickname: 'player',
      rounds: 1,
      letsGame: false,
      gameState: 'configure-game',
      score: 0,
      attempingResults: false,
      showResults: false,
      result: null,
      finalScore: false,
      results: [],
      newArray: [],
    })
  }

  render() {
    const {
      nickname,
      letsGame,
      gameState,
      results,
      attempingResults,
      score,
      showResults,
      result,
      newArray,
      finalScore,
      rounds,
    } = this.state
    return (
      <>
        {!letsGame && (
          <>
            {gameState === 'fatal-error' && (
              <SignupContainer>
                <HighLightName>
                  ROCK
                  <br />
                  PAPER
                  <br />
                  SCISSORS
                </HighLightName>
                <HighLightName>
                  <br />
                  FATAL ERROR OCCURS. PLEASE, REFRESH THIS PAGE
                </HighLightName>
              </SignupContainer>
            )}
            {gameState === 'waiting-game' && (
              <SignupContainer>
                <HighLightName>
                  ROCK
                  <br /> PAPER <br /> SCISSORS
                </HighLightName>
                <HighLightName>
                  <br />
                  WAITING FOR GAME SETUP
                </HighLightName>
              </SignupContainer>
            )}
            {(gameState === 'join-game' || gameState === 'configure-game') && (
              <SignupContainer>
                <HighLightName>
                  ROCK
                  <br /> PAPER <br /> SCISSORS
                </HighLightName>
                <br />
                <br />
                <HighLightName> Nickname </HighLightName>
                <input
                  type="text"
                  name="nickname"
                  value={nickname}
                  onChange={this.nicknameChanges}
                />
                {gameState === 'configure-game' && (
                  <>
                    <br />
                    <HighLightName> Rounds </HighLightName>
                    <input
                      type="number"
                      name="rounds"
                      min="1"
                      max="10"
                      value={rounds}
                      onChange={this.roundChanges}
                    />
                  </>
                )}
                <br />
                <br />
                <PlayButton
                  type="image"
                  name="continue"
                  alt="play"
                  onClick={this.play}
                >
                  <PlayImage src="https://freeiconshop.com/wp-content/uploads/edd/play-rounded-outline.png" />
                </PlayButton>
              </SignupContainer>
            )}
            {(gameState === 'waiting-for-restarting-game' ||
              gameState === 'show-final-score') && (
              <SignupContainer>
                <HighLightTitle>ROUND&apos;S SCORE</HighLightTitle>
                {Object.keys(results.finalScore).map(index => (
                  <>
                    <HighLightName>
                      <br />
                      ROUND {parseInt(index) + 1}
                    </HighLightName>
                    <TableCentered>
                      <tbody>
                        <tr key={results.finalScore[index].idx}>
                          {Object.keys(results.finalScore[index].choices).map(
                            choice => (
                              <td>
                                <table>
                                  <tbody>
                                    <tr key="tr-{idx}-1">
                                      <td key="td-{idx}-1">
                                        <HighLightName>{choice}</HighLightName>
                                      </td>
                                    </tr>
                                    <tr key="tr-{idx}-2">
                                      <td key="td-{idx}-2">
                                        <ScoreImage
                                          src={
                                            this.choicesList[
                                              results.finalScore[index].choices[
                                                choice
                                              ]
                                            ].imageUrl
                                          }
                                          alt={
                                            this.choicesList[
                                              results.finalScore[index].choices[
                                                choice
                                              ]
                                            ].id
                                          }
                                          key={
                                            this.choicesList[
                                              results.finalScore[index].choices[
                                                choice
                                              ]
                                            ].id
                                          }
                                        />
                                      </td>
                                    </tr>
                                    <tr key="tr-{idx}-3">
                                      <td key="td-{idx}-3">
                                        <HighLightName>
                                          {results.finalScore[index]
                                            .roundResults[choice] && <> WON</>}
                                          {results.finalScore[index]
                                            .roundResults[choice] != null &&
                                            !results.finalScore[index]
                                              .roundResults[choice] && (
                                              <> LOST</>
                                            )}
                                          {results.finalScore[index]
                                            .roundResults[choice] == null && (
                                            <>IT IS DRAW</>
                                          )}
                                        </HighLightName>
                                      </td>
                                    </tr>
                                    <tr key="tr-{idx}-4">
                                      <td key="td-{idx}-4">
                                        <HighLightName>
                                          {
                                            results.finalScore[index].scores[
                                              choice
                                            ]
                                          }
                                        </HighLightName>
                                      </td>
                                    </tr>
                                  </tbody>
                                </table>
                              </td>
                            ),
                          )}
                        </tr>
                      </tbody>
                    </TableCentered>
                  </>
                ))}
                <HighLightTitle>FINAL SCORE</HighLightTitle>
                <TableCentered>
                  <tbody>
                    <tr>
                      {Object.keys(results.finalScores).map(key => (
                        <td key={key}>
                          <table>
                            <tbody>
                              <tr key="tr-{idx}{key}-1">
                                <td key="td-{idx}{key}-1">
                                  <HighLightName>{key}</HighLightName>
                                </td>
                              </tr>
                              <tr key="tr-{idx}{key}-2">
                                <td key="td-{idx}{key}-2">
                                  <HighLightName>
                                    {results.finalScores[key]}
                                  </HighLightName>
                                </td>
                              </tr>
                            </tbody>
                          </table>
                        </td>
                      ))}
                    </tr>
                  </tbody>
                </TableCentered>
                {gameState !== 'waiting-for-restarting-game' && (
                  <button type="button" onClick={this.restartGame}>
                    Reset game
                  </button>
                )}
                {gameState === 'waiting-for-restarting-game' && (
                  <HighLightTitle>
                    WAITING WHILE ALL PLAYERS ARE READY FOR GAMING AGAIN...
                  </HighLightTitle>
                )}
              </SignupContainer>
            )}
          </>
        )}
        {letsGame && (
          <RpsPage
            choicesList={this.choicesList}
            sendChoice={this.sendChoice}
            score={score}
            attempingResults={attempingResults}
            playAgain={this.playAgain}
            showResults={showResults}
            result={result}
            newArray={newArray}
            finalScore={finalScore}
            showFinalScore={this.showFinalScore}
          />
        )}
      </>
    )
  }
}

export default Signup
