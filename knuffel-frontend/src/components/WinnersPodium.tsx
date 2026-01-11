"use client"

import React from "react"
import BunnyMascot from "./BunnyMascot"

interface Player {
    id: string
    username: string
    score: number
}

interface WinnersPodiumProps {
    players: Player[]
}

const WinnersPodium: React.FC<WinnersPodiumProps> = ({ players }) => {
    // Sort players by score descending
    const sortedPlayers = [...players].sort((a, b) => b.score - a.score)
    const podium = [
        sortedPlayers[1], // 2nd place (left)
        sortedPlayers[0], // 1st place (center)
        sortedPlayers[2], // 3rd place (right)
    ]

    return (
        <div className="flex items-end justify-center gap-4 mt-8 mb-4 h-[300px]">
            {/* 2nd Place */}
            {podium[0] && (
                <div className="flex flex-col items-center">
                    <div className="mb-2 text-center animate-bounce delay-100 flex flex-col items-center w-full">
                        <div className="w-12 h-12 bg-pink-100 rounded-full flex items-center justify-center font-bold text-pink-600 border-2 border-pink-300 shrink-0">
                            {podium[0].username.charAt(0).toUpperCase()}
                        </div>
                        <p className="text-sm font-bold text-gray-700 mt-1 truncate w-full max-w-[100px]">{podium[0].username}</p>
                    </div>
                    <div className="w-24 h-24 bg-gradient-to-t from-gray-300 to-gray-100 rounded-t-2xl border-x-4 border-t-4 border-gray-400 flex flex-col items-center justify-center shadow-lg">
                        <span className="text-3xl font-black text-gray-500">2</span>
                        <span className="text-xs font-bold text-gray-500 uppercase">{podium[0].score} Pkt.</span>
                    </div>
                </div>
            )}

            {/* 1st Place */}
            {podium[1] && (
                <div className="flex flex-col items-center">
                    <div className="mb-4 text-center animate-bounce flex flex-col items-center w-full">
                        <div className="w-16 h-16 bg-yellow-100 rounded-full flex items-center justify-center font-bold text-yellow-600 border-2 border-yellow-400 -mt-2 shadow-inner shrink-0">
                            {podium[1].username.charAt(0).toUpperCase()}
                        </div>
                        <p className="text-lg font-black text-pink-600 mt-1 drop-shadow-sm truncate w-full max-w-[140px]">{podium[1].username}</p>
                    </div>
                    <div className="w-32 h-36 bg-gradient-to-t from-yellow-300 to-yellow-100 rounded-t-2xl border-x-4 border-t-4 border-yellow-400 flex flex-col items-center justify-center shadow-xl relative">
                        <div className="absolute -top-6 text-4xl animate-pulse">👑</div>
                        <span className="text-5xl font-black text-yellow-600">1</span>
                        <span className="text-sm font-black text-yellow-600 uppercase">{podium[1].score} Pkt.</span>
                    </div>
                </div>
            )}

            {/* 3rd Place */}
            {podium[2] && (
                <div className="flex flex-col items-center">
                    <div className="mb-2 text-center animate-bounce delay-200 flex flex-col items-center w-full">
                        <div className="w-12 h-12 bg-orange-100 rounded-full flex items-center justify-center font-bold text-orange-600 border-2 border-orange-300 shrink-0">
                            {podium[2].username.charAt(0).toUpperCase()}
                        </div>
                        <p className="text-sm font-bold text-gray-700 mt-1 truncate w-full max-w-[100px]">{podium[2].username}</p>
                    </div>
                    <div className="w-24 h-16 bg-gradient-to-t from-orange-300 to-orange-100 rounded-t-2xl border-x-4 border-t-4 border-orange-400 flex flex-col items-center justify-center shadow-lg">
                        <span className="text-3xl font-black text-orange-500">3</span>
                        <span className="text-xs font-bold text-orange-500 uppercase">{podium[2].score} Pkt.</span>
                    </div>
                </div>
            )}
        </div>
    )
}

export default WinnersPodium
