"use client"

import type React from "react"
import { useEffect, useState } from "react"
import { useParams, useNavigate } from "react-router-dom"
import BunnyMascot from "../components/BunnyMascot"
import AnimatedDiceBackground from "../components/AnimatedDiceBackground"
import Dice from "../components/Dice"
import ScoreCard from "../components/ScoreCard"
import WinnersPodium from "../components/WinnersPodium"
import { rollDice, keepDice, selectField, getGameState } from "../services/api"
import { connectToGameSSE, type SSEEvent } from "../services/sse"

interface DiceState {
  value: number
  kept: boolean
}

interface Player {
  id: string
  username: string
  score: number
}

interface GameState {
  gameId: string
  currentPlayerId: string
  players: Player[]
  dice: DiceState[]
  rollsLeft: number
  scores: any[] // TODO: Define proper score interface
  gameOver: boolean
  winner?: Player
}

const GamePage: React.FC = () => {
  const { gameId } = useParams<{ gameId: string }>()
  const navigate = useNavigate()
  const [gameState, setGameState] = useState<GameState | null>(null)
  const [loading, setLoading] = useState(true)
  const [rolling, setRolling] = useState(false)
  const [error, setError] = useState("")
  const [currentUserId, setCurrentUserId] = useState("mock-user-456") // MOCK for testing

  useEffect(() => {
    if (!gameId) return

    // Fetch initial game state
    const fetchGameState = async () => {
      try {
        const data = await getGameState(gameId)
        setGameState(data)
        setLoading(false)
      } catch (err: any) {
        setError(err.response?.data?.message || "Fehler beim Laden des Spiels")
        setLoading(false)
      }
    }

    fetchGameState()

    // Connect to SSE for real-time updates
    const eventSource = connectToGameSSE(gameId, (event: SSEEvent) => {
      if (event.type === "game_updated") {
        setGameState(event.data)
      }
    })

    return () => {
      eventSource.close()
    }
  }, [gameId])

  const handleRollDice = async () => {
    if (!gameId || !gameState || gameState.rollsLeft <= 0) return

    setRolling(true)
    setError("")

    try {
      const updatedState = await rollDice(gameId)
      setGameState(updatedState)
      // SSE will update game state as well, but this ensures immediate mock update
      setTimeout(() => setRolling(false), 600)
    } catch (err: any) {
      setError(err.response?.data?.message || "Fehler beim Würfeln")
      setRolling(false)
    }
  }

  const handleToggleDice = async (index: number) => {
    if (!gameId || !gameState || gameState.rollsLeft === 3 || rolling) return

    const currentKept = gameState.dice[index].kept

    try {
      const updatedState = await keepDice(gameId, index, !currentKept)
      setGameState(updatedState)
      // SSE will update game state
    } catch (err: any) {
      setError(err.response?.data?.message || "Fehler beim Behalten des Würfels")
    }
  }

  const handleSelectField = async (fieldName: string) => {
    if (!gameId) return

    setError("")

    try {
      console.log(`GamePage: handleSelectField aufgerufen für ${fieldName}`)
      const updatedState = await selectField(gameId, fieldName)
      setGameState(updatedState)
      // SSE will update game state
    } catch (err: any) {
      setError(err.response?.data?.message || "Fehler beim Auswählen des Feldes")
    }
  }

  const handleQuitGame = () => {
    navigate("/")
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-pink-100 via-pink-200 to-rose-200 flex items-center justify-center">
        <div className="text-2xl text-pink-600 font-bold">Lädt Spiel...</div>
      </div>
    )
  }

  if (error && !gameState) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-pink-100 via-pink-200 to-rose-200 flex items-center justify-center">
        <div className="bg-white/90 rounded-3xl p-8 border-4 border-red-300">
          <p className="text-red-600 font-bold mb-4">{error}</p>
          <button
            onClick={() => navigate("/")}
            className="bg-pink-500 hover:bg-pink-600 text-white font-bold py-2 px-6 rounded-xl"
          >
            Zurück
          </button>
        </div>
      </div>
    )
  }

  const isMyTurn = gameState && currentUserId === gameState.currentPlayerId
  const canRoll = isMyTurn && (gameState?.rollsLeft || 0) > 0
  const canSelectField = isMyTurn && (gameState?.rollsLeft || 3) < 3

  return (
    <div className="min-h-screen bg-gradient-to-br from-pink-100 via-pink-200 to-rose-200 relative overflow-hidden">
      <AnimatedDiceBackground />

      <div className="relative z-10 p-4 max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-4">
            <BunnyMascot size="md" />
            <div>
              <h1 className="text-3xl font-bold text-pink-600">Knuffel Spiel</h1>
              <p className="text-pink-700">
                {isMyTurn
                  ? "Du bist dran!"
                  : `${gameState?.players.find((p) => p.id === gameState.currentPlayerId)?.username} ist dran`}
              </p>
            </div>
          </div>
          <button
            onClick={handleQuitGame}
            className="bg-gray-500 hover:bg-gray-600 text-white font-bold py-2 px-4 rounded-xl transition-all"
          >
            Spiel verlassen
          </button>
        </div>

        {/* Game Over Screen */}
        {gameState?.gameOver && (
          <div className="fixed inset-0 bg-black/70 backdrop-blur-md flex items-center justify-center z-50 p-4">
            <div className="bg-white rounded-[3rem] p-10 border-8 border-pink-300 text-center max-w-2xl shadow-[0_0_50px_rgba(236,72,153,0.3)] animate-fadeIn">
              <h2 className="text-5xl font-black text-pink-600 mb-2 uppercase tracking-tighter">Spiel beendet!</h2>
              <p className="text-pink-400 font-bold mb-6">Was für eine Runde!</p>

              <WinnersPodium players={gameState.players} />

              <div className="flex flex-col sm:flex-row gap-4 justify-center mt-10">
                <button
                  onClick={() => window.location.reload()}
                  className="bg-gradient-to-r from-pink-500 to-rose-500 hover:from-pink-600 hover:to-rose-600 text-white font-black py-4 px-10 rounded-2xl shadow-lg hover:shadow-pink-200 transition-all transform hover:scale-105 uppercase tracking-wider"
                >
                  Neues Spiel
                </button>
                <button
                  onClick={handleQuitGame}
                  className="bg-gray-200 hover:bg-gray-300 text-gray-600 font-bold py-4 px-10 rounded-2xl transition-all"
                >
                  Zum Menü
                </button>
              </div>
            </div>
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-8 gap-8">
          {/* Left: Player List */}
          <div className="lg:col-span-3 bg-white/90 backdrop-blur-sm rounded-3xl p-6 border-4 border-pink-300 shadow-xl h-fit">
            <h2 className="text-xl font-black text-pink-600 mb-6 uppercase tracking-widest px-2">Spieler</h2>
            <div className="space-y-4">
              {gameState?.players.map((player) => (
                <div
                  key={player.id}
                  className={`
                    p-4 rounded-2xl border-2 transition-all duration-300 w-full overflow-hidden
                    ${player.id === gameState.currentPlayerId ? "bg-pink-100 border-pink-400 ring-4 ring-pink-200 scale-[1.02]" : "bg-pink-50 border-pink-200 opacity-80"}
                  `}
                >
                  <div className="flex items-center justify-between gap-3 w-full">
                    <div className="flex items-center gap-3 min-w-0 flex-1">
                      <div className={`w-12 h-12 rounded-full flex items-center justify-center font-black text-white shadow-inner shrink-0 ${player.id === gameState.currentPlayerId ? "bg-pink-500 animate-pulse" : "bg-pink-300"}`}>
                        {player.username.charAt(0).toUpperCase()}
                      </div>
                      <div className="flex flex-col min-w-0 flex-1 text-left">
                        <span className="font-bold text-gray-800 leading-tight truncate block" title={player.username}>
                          {player.username}
                        </span>
                        {player.id === gameState.currentPlayerId && (
                          <span className="text-[10px] font-black text-pink-500 uppercase tracking-tighter truncate block">
                            Am Zug
                          </span>
                        )}
                      </div>
                    </div>
                    <div className="flex flex-col items-end shrink-0 min-w-fit ml-auto">
                      <span className="text-2xl font-black text-pink-600 leading-tight">{player.score}</span>
                      <span className="text-[10px] font-bold text-pink-400 uppercase">Punkte</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>

            {/* Developer Trigger - Temporary for testing */}
            <div className="mt-8 border-t border-pink-100 pt-4 px-2">
              <button
                onClick={() => handleSelectField("debug_game_over")}
                className="text-[10px] text-pink-300 hover:text-pink-500 font-bold uppercase tracking-widest transition-colors w-full text-center"
              >
                Ende simulieren (Dev)
              </button>
            </div>
          </div>

          {/* Center/Right: Game Area + Score Card */}
          <div className="lg:col-span-5 space-y-8">
            {/* Dice Area */}
            <div className="bg-white/95 backdrop-blur-md rounded-[2.5rem] p-8 border-4 border-pink-300 shadow-2xl relative overflow-hidden">
              <div className="absolute top-0 right-0 p-8 opacity-10 pointer-events-none">
                <BunnyMascot size="lg" />
              </div>

              <div className="flex items-center justify-between mb-8">
                <div>
                  <h2 className="text-2xl font-black text-pink-600 uppercase tracking-wider">Deine Würfel</h2>
                  <p className="text-pink-400 font-bold">Wähle die Würfel, die du behalten willst</p>
                </div>
                <div className="bg-gradient-to-br from-pink-500 to-rose-500 p-[2px] rounded-2xl shadow-lg">
                  <div className="bg-white px-6 py-3 rounded-[14px]">
                    <span className="text-sm font-black text-gray-400 uppercase tracking-widest mr-2">Versuche:</span>
                    <span className="text-2xl font-black text-pink-600 tracking-tighter">{gameState?.rollsLeft || 0}</span>
                    <span className="text-lg font-bold text-pink-300">/3</span>
                  </div>
                </div>
              </div>

              {/* Dice */}
              <div className="flex flex-wrap justify-center gap-6 mb-10">
                {gameState?.dice.map((die, index) => (
                  <Dice
                    key={index}
                    value={die.value}
                    kept={die.kept}
                    onClick={() => handleToggleDice(index)}
                    disabled={!isMyTurn || (gameState?.rollsLeft || 0) === 3}
                    rolling={rolling}
                  />
                ))}
              </div>

              {/* Roll Button */}
              <button
                onClick={handleRollDice}
                disabled={!canRoll || rolling}
                className={`
                  w-full py-6 px-10 rounded-[1.5rem] font-black text-2xl shadow-2xl transition-all relative overflow-hidden group
                  ${canRoll && !rolling
                    ? "bg-gradient-to-r from-pink-500 via-rose-500 to-pink-600 text-white hover:scale-[1.02] hover:shadow-pink-200 active:scale-[0.98]"
                    : "bg-gray-100 text-gray-400 cursor-not-allowed"}
                `}
              >
                <span className="relative z-10 uppercase tracking-widest">
                  {rolling ? "Die Würfel rollen..." : (gameState?.rollsLeft === 3 ? "Erster Wurf!" : "Nochmal würfeln!")}
                </span>
                {canRoll && !rolling && (
                  <div className="absolute inset-0 bg-gradient-to-r from-white/0 via-white/20 to-white/0 translate-x-[-100%] group-hover:translate-x-[100%] transition-transform duration-1000"></div>
                )}
              </button>

              {/* Instructions */}
              {isMyTurn && (
                <div className="mt-6 text-center">
                  <p className="text-sm font-bold text-gray-500 bg-gray-50 inline-block px-6 py-2 rounded-full border border-gray-100">
                    {(gameState?.rollsLeft || 0) === 3 && 'Klicke auf den Button zum Starten!'}
                    {(gameState?.rollsLeft || 0) < 3 &&
                      (gameState?.rollsLeft || 0) > 0 &&
                      "Behalte gute Würfel und würfle den Rest neu!"}
                    {(gameState?.rollsLeft || 0) === 0 && "Klasse! Wähle jetzt ein Feld im Punkteblock unten."}
                  </p>
                </div>
              )}
            </div>

            {/* Error Message */}
            {error && <div className="p-4 bg-red-50 border-2 border-red-200 rounded-2xl text-red-600 font-bold animate-wiggle text-center shadow-lg">{error}</div>}

            {/* Score Card - MOVED HERE */}
            <ScoreCard scores={gameState?.scores || []} onSelectField={handleSelectField} disabled={!canSelectField} />
          </div>
        </div>
      </div>
    </div>
  )
}

export default GamePage
