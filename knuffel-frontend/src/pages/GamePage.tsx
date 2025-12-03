"use client"

import type React from "react"
import { useEffect, useState } from "react"
import { useParams, useNavigate } from "react-router-dom"
import BunnyMascot from "../components/BunnyMascot"
import AnimatedDiceBackground from "../components/AnimatedDiceBackground"
import Dice from "../components/Dice"
import ScoreCard from "../components/ScoreCard"
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
  const [currentUserId, setCurrentUserId] = useState("") // TODO: Get from auth context

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
      await rollDice(gameId)
      // SSE will update game state
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
      await keepDice(gameId, index, !currentKept)
      // SSE will update game state
    } catch (err: any) {
      setError(err.response?.data?.message || "Fehler beim Behalten des Würfels")
    }
  }

  const handleSelectField = async (fieldName: string) => {
    if (!gameId) return

    setError("")

    try {
      await selectField(gameId, fieldName)
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
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-white rounded-3xl p-8 border-4 border-pink-300 text-center max-w-md">
              <BunnyMascot size="lg" className="mx-auto mb-4" />
              <h2 className="text-4xl font-bold text-pink-600 mb-4">Spiel beendet!</h2>
              <p className="text-2xl font-bold text-gray-800 mb-6">Gewinner: {gameState.winner?.username}</p>
              <button
                onClick={handleQuitGame}
                className="bg-gradient-to-r from-pink-500 to-rose-500 hover:from-pink-600 hover:to-rose-600 text-white font-bold py-3 px-8 rounded-xl"
              >
                Zurück zur Startseite
              </button>
            </div>
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Left: Player List */}
          <div className="bg-white/90 backdrop-blur-sm rounded-2xl p-6 border-4 border-pink-300 shadow-xl h-fit">
            <h2 className="text-xl font-bold text-pink-600 mb-4">Spieler</h2>
            <div className="space-y-3">
              {gameState?.players.map((player) => (
                <div
                  key={player.id}
                  className={`
                    p-4 rounded-xl border-2 transition-all
                    ${player.id === gameState.currentPlayerId ? "bg-pink-100 border-pink-400 ring-2 ring-pink-300" : "bg-pink-50 border-pink-200"}
                  `}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-pink-400 rounded-full flex items-center justify-center font-bold text-white">
                        {player.username.charAt(0).toUpperCase()}
                      </div>
                      <span className="font-semibold text-gray-800">{player.username}</span>
                    </div>
                    <span className="font-bold text-pink-600">{player.score}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Center: Game Area */}
          <div className="space-y-6">
            {/* Dice Area */}
            <div className="bg-white/90 backdrop-blur-sm rounded-2xl p-6 border-4 border-pink-300 shadow-xl">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-xl font-bold text-pink-600">Würfel</h2>
                <div className="bg-pink-100 px-4 py-2 rounded-xl border-2 border-pink-300">
                  <span className="font-bold text-pink-700">Würfe übrig: {gameState?.rollsLeft || 0}</span>
                </div>
              </div>

              {/* Dice */}
              <div className="flex justify-center gap-4 mb-6">
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
                className="w-full bg-gradient-to-r from-pink-500 to-rose-500 hover:from-pink-600 hover:to-rose-600 text-white font-bold py-4 px-6 rounded-xl shadow-lg hover:shadow-xl transform hover:scale-105 transition-all disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none"
              >
                {rolling ? "Würfelt..." : "Würfeln!"}
              </button>

              {/* Instructions */}
              {isMyTurn && (
                <p className="text-center text-sm text-gray-600 mt-4">
                  {(gameState?.rollsLeft || 0) === 3 && 'Klicke auf "Würfeln" um zu starten!'}
                  {(gameState?.rollsLeft || 0) < 3 &&
                    (gameState?.rollsLeft || 0) > 0 &&
                    "Klicke auf Würfel um sie zu behalten, dann würfel erneut oder wähle ein Feld."}
                  {(gameState?.rollsLeft || 0) === 0 && "Wähle ein Feld für deine Punkte!"}
                </p>
              )}
            </div>

            {/* Error Message */}
            {error && <div className="p-4 bg-red-100 border-2 border-red-300 rounded-xl text-red-700">{error}</div>}
          </div>

          {/* Right: Score Card */}
          <ScoreCard scores={gameState?.scores || []} onSelectField={handleSelectField} disabled={!canSelectField} />
        </div>
      </div>
    </div>
  )
}

export default GamePage
