"use client"

import type React from "react"
import { useEffect, useState } from "react"
import { useParams, useNavigate } from "react-router-dom"
import BunnyMascot from "../components/BunnyMascot"
import AnimatedDiceBackground from "../components/AnimatedDiceBackground"
import { getLobby, startGame, kickPlayer } from "../services/api"
import { connectToLobbySSE, type SSEEvent } from "../services/sse"

interface Player {
  id: string
  username: string
  isLeader: boolean
}

interface LobbyData {
  lobbyId: string
  joinCode: string
  players: Player[]
  leaderId: string
}

const LobbyPage: React.FC = () => {
  const { lobbyId } = useParams<{ lobbyId: string }>()
  const navigate = useNavigate()
  const [lobby, setLobby] = useState<LobbyData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")
  const [copied, setCopied] = useState(false)
  const [currentUserId, setCurrentUserId] = useState("") // TODO: Get from auth context

  useEffect(() => {
    if (!lobbyId) return

    // Fetch initial lobby data
    const fetchLobby = async () => {
      try {
        const data = await getLobby(lobbyId)
        setLobby(data)
        setLoading(false)
      } catch (err: any) {
        setError(err.response?.data?.message || "Fehler beim Laden der Lobby")
        setLoading(false)
      }
    }

    fetchLobby()

    // Connect to SSE for real-time updates
    const eventSource = connectToLobbySSE(lobbyId, (event: SSEEvent) => {
      switch (event.type) {
        case "player_joined":
          // TODO: Update player list
          console.log("Player joined:", event.data)
          break
        case "player_left":
          // TODO: Update player list
          console.log("Player left:", event.data)
          break
        case "player_kicked":
          // TODO: Handle player kicked
          console.log("Player kicked:", event.data)
          break
        case "leader_changed":
          // TODO: Update leader
          console.log("Leader changed:", event.data)
          break
        case "game_started":
          // Navigate to game
          navigate(`/game/${event.data.gameId}`)
          break
      }
    })

    return () => {
      eventSource.close()
    }
  }, [lobbyId, navigate])

  const handleCopyCode = () => {
    if (lobby) {
      navigator.clipboard.writeText(lobby.joinCode)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  const handleStartGame = async () => {
    if (!lobbyId || !lobby) return

    if (lobby.players.length < 2) {
      setError("Mindestens 2 Spieler benötigt")
      return
    }

    try {
      await startGame(lobbyId)
      // SSE event will handle navigation
    } catch (err: any) {
      setError(err.response?.data?.message || "Fehler beim Starten des Spiels")
    }
  }

  const handleKickPlayer = async (playerId: string) => {
    if (!lobbyId) return

    try {
      await kickPlayer(lobbyId, playerId)
      // SSE event will update player list
    } catch (err: any) {
      setError(err.response?.data?.message || "Fehler beim Kicken des Spielers")
    }
  }

  const handleLeaveLobby = () => {
    navigate("/")
  }

  const isLeader = lobby && currentUserId === lobby.leaderId

  if (loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-pink-100 via-pink-200 to-rose-200 flex items-center justify-center">
        <div className="text-2xl text-pink-600 font-bold">Lädt...</div>
      </div>
    )
  }

  if (error && !lobby) {
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

  return (
    <div className="min-h-screen bg-gradient-to-br from-pink-100 via-pink-200 to-rose-200 relative overflow-hidden">
      <AnimatedDiceBackground />

      <div className="relative z-10 flex flex-col items-center justify-center min-h-screen p-4">
        {/* Header with Mascot */}
        <div className="flex items-center gap-4 mb-8">
          <BunnyMascot size="md" />
          <h1 className="text-4xl font-bold text-pink-600">Lobby Warteraum</h1>
        </div>

        {/* Main Card */}
        <div className="bg-white/90 backdrop-blur-sm rounded-3xl shadow-2xl p-8 w-full max-w-2xl border-4 border-pink-300">
          {/* Join Code */}
          <div className="mb-6 p-6 bg-gradient-to-r from-pink-500 to-rose-500 rounded-2xl text-white">
            <p className="text-sm font-semibold mb-2">Lobby-Code:</p>
            <div className="flex items-center justify-between">
              <span className="text-4xl font-bold tracking-widest">{lobby?.joinCode}</span>
              <button
                onClick={handleCopyCode}
                className="bg-white text-pink-600 font-bold py-2 px-4 rounded-xl hover:bg-pink-50 transition-all"
              >
                {copied ? "✓ Kopiert!" : "Kopieren"}
              </button>
            </div>
          </div>

          {/* Player List */}
          <div className="mb-6">
            <h2 className="text-xl font-bold text-gray-800 mb-4">Spieler ({lobby?.players.length || 0})</h2>
            <div className="space-y-2">
              {lobby?.players.map((player) => (
                <div
                  key={player.id}
                  className="flex items-center justify-between p-4 bg-pink-50 rounded-xl border-2 border-pink-200"
                >
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 bg-pink-300 rounded-full flex items-center justify-center font-bold text-white">
                      {player.username.charAt(0).toUpperCase()}
                    </div>
                    <span className="font-semibold text-gray-800">{player.username}</span>
                    {player.isLeader && (
                      <span className="bg-yellow-400 text-yellow-900 text-xs font-bold px-2 py-1 rounded-full">
                        👑 Leiter
                      </span>
                    )}
                  </div>
                  {isLeader && !player.isLeader && (
                    <button
                      onClick={() => handleKickPlayer(player.id)}
                      className="bg-red-500 hover:bg-red-600 text-white font-bold py-1 px-3 rounded-lg text-sm transition-all"
                    >
                      Kicken
                    </button>
                  )}
                </div>
              ))}
            </div>
          </div>

          {/* Error Message */}
          {error && (
            <div className="mb-4 p-3 bg-red-100 border-2 border-red-300 rounded-xl text-red-700 text-sm">{error}</div>
          )}

          {/* Action Buttons */}
          <div className="flex gap-4">
            {isLeader && (
              <button
                onClick={handleStartGame}
                disabled={(lobby?.players.length || 0) < 2}
                className="flex-1 bg-gradient-to-r from-green-500 to-emerald-500 hover:from-green-600 hover:to-emerald-600 text-white font-bold py-4 px-6 rounded-xl shadow-lg hover:shadow-xl transform hover:scale-105 transition-all disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none"
              >
                Spiel starten
              </button>
            )}
            <button
              onClick={handleLeaveLobby}
              className={`${isLeader ? "flex-none" : "flex-1"} bg-gray-500 hover:bg-gray-600 text-white font-bold py-4 px-6 rounded-xl shadow-lg hover:shadow-xl transform hover:scale-105 transition-all`}
            >
              Lobby verlassen
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default LobbyPage
