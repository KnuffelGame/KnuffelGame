"use client"

import type React from "react"
import { useState } from "react"
import { useNavigate } from "react-router-dom"
import BunnyMascot from "../components/BunnyMascot"
import AnimatedDiceBackground from "../components/AnimatedDiceBackground"
import { createGuest, createLobby, joinLobby } from "../services/api"

const HomePage: React.FC = () => {
  const [username, setUsername] = useState("")
  const [joinCode, setJoinCode] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const navigate = useNavigate()

  const validateUsername = (name: string): boolean => {
    return name.length >= 3 && name.length <= 20
  }

  const handleCreateLobby = async () => {
    if (!validateUsername(username)) {
      setError("Username muss zwischen 3 und 20 Zeichen lang sein")
      return
    }

    setLoading(true)
    setError("")

    try {
      // Create guest user first
      await createGuest(username)

      // Then create lobby
      const lobbyData = await createLobby()

      // Navigate to lobby
      navigate(`/lobby/${lobbyData.lobbyId}`)
    } catch (err: any) {
      setError(err.response?.data?.message || "Fehler beim Erstellen der Lobby")
      console.error("Error creating lobby:", err)
    } finally {
      setLoading(false)
    }
  }

  const handleJoinLobby = async () => {
    if (!validateUsername(username)) {
      setError("Username muss zwischen 3 und 20 Zeichen lang sein")
      return
    }

    if (!joinCode || joinCode.length < 4) {
      setError("Bitte gib einen gültigen Join-Code ein")
      return
    }

    setLoading(true)
    setError("")

    try {
      // Create guest user first
      await createGuest(username)

      // Then join lobby
      const lobbyData = await joinLobby(joinCode)

      // Navigate to lobby
      navigate(`/lobby/${lobbyData.lobbyId}`)
    } catch (err: any) {
      setError(err.response?.data?.message || "Fehler beim Beitreten der Lobby")
      console.error("Error joining lobby:", err)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-pink-100 via-pink-200 to-rose-200 relative overflow-hidden">
      <AnimatedDiceBackground />

      <div className="relative z-10 flex flex-col items-center justify-center min-h-screen p-4">
        {/* Logo and Mascot */}
        <div className="flex flex-col items-center mb-8 animate-[fadeIn_0.8s_ease-in]">
          <BunnyMascot size="lg" className="mb-4" />
          <h1 className="text-6xl font-bold text-pink-600 mb-2 text-balance text-center">Knuffel</h1>
        </div>

        {/* Main Card */}
        <div className="bg-white/90 backdrop-blur-sm rounded-3xl shadow-2xl p-8 w-full max-width-md border-4 border-pink-300">
          {/* Username Input */}
          <div className="mb-6">
            <label htmlFor="username" className="block text-sm font-semibold text-gray-700 mb-2">
              Dein Name
            </label>
            <input
              id="username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="Gib deinen Namen ein..."
              className="w-full px-4 py-3 rounded-xl border-2 border-pink-300 focus:border-pink-500 focus:outline-none focus:ring-2 focus:ring-pink-200 transition-all"
              minLength={3}
              maxLength={20}
              disabled={loading}
            />
            <p className="text-xs text-gray-500 mt-1">3-20 Zeichen</p>
          </div>

          {/* Error Message */}
          {error && (
            <div className="mb-4 p-3 bg-red-100 border-2 border-red-300 rounded-xl text-red-700 text-sm">{error}</div>
          )}

          {/* Create Lobby Button */}
          <button
            onClick={handleCreateLobby}
            disabled={loading || !validateUsername(username)}
            className="w-full bg-gradient-to-r from-pink-500 to-rose-500 hover:from-pink-600 hover:to-rose-600 text-white font-bold py-4 px-6 rounded-xl shadow-lg hover:shadow-xl transform hover:scale-105 transition-all disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none mb-4"
          >
            {loading ? "Lädt..." : "Neue Lobby erstellen"}
          </button>

          {/* Divider */}
          <div className="flex items-center my-6">
            <div className="flex-1 border-t-2 border-gray-300"></div>
            <span className="px-4 text-gray-500 font-semibold">oder</span>
            <div className="flex-1 border-t-2 border-gray-300"></div>
          </div>

          {/* Join Lobby */}
          <div className="mb-4">
            <label htmlFor="joinCode" className="block text-sm font-semibold text-gray-700 mb-2">
              Lobby-Code
            </label>
            <input
              id="joinCode"
              type="text"
              value={joinCode}
              onChange={(e) => setJoinCode(e.target.value.toUpperCase())}
              placeholder="ABCD"
              className="w-full px-4 py-3 rounded-xl border-2 border-pink-300 focus:border-pink-500 focus:outline-none focus:ring-2 focus:ring-pink-200 transition-all uppercase"
              disabled={loading}
            />
          </div>

          <button
            onClick={handleJoinLobby}
            disabled={loading || !validateUsername(username) || !joinCode}
            className="w-full bg-gradient-to-r from-purple-500 to-pink-500 hover:from-purple-600 hover:to-pink-600 text-white font-bold py-4 px-6 rounded-xl shadow-lg hover:shadow-xl transform hover:scale-105 transition-all disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none"
          >
            {loading ? "Lädt..." : "Lobby beitreten"}
          </button>
        </div>

        {/* Footer */}
      
      </div>
    </div>
  )
}

export default HomePage
