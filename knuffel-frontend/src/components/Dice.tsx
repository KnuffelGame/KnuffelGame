"use client"

import type React from "react"
import type { JSX } from "react/jsx-runtime" // Import JSX to fix the undeclared variable error

interface DiceProps {
  value: number
  kept: boolean
  onClick?: () => void
  disabled?: boolean
  rolling?: boolean
}

const Dice: React.FC<DiceProps> = ({ value, kept, onClick, disabled = false, rolling = false }) => {
  const getDotPositions = (num: number): JSX.Element[] => {
    const dots: JSX.Element[] = []
    const positions = {
      1: [[50, 50]],
      2: [
        [30, 30],
        [70, 70],
      ],
      3: [
        [30, 30],
        [50, 50],
        [70, 70],
      ],
      4: [
        [30, 30],
        [70, 30],
        [30, 70],
        [70, 70],
      ],
      5: [
        [30, 30],
        [70, 30],
        [50, 50],
        [30, 70],
        [70, 70],
      ],
      6: [
        [30, 30],
        [70, 30],
        [30, 50],
        [70, 50],
        [30, 70],
        [70, 70],
      ],
    }

    const coords = positions[num as keyof typeof positions] || []
    coords.forEach((coord, idx) => {
      dots.push(<circle key={idx} cx={coord[0]} cy={coord[1]} r="8" fill="white" />)
    })

    return dots
  }

  return (
    <button
      onClick={onClick}
      disabled={disabled || rolling}
      className={`
        relative w-20 h-20 transition-all duration-300 transform
        ${rolling ? "animate-spin" : ""}
        ${kept ? "scale-110 shadow-2xl" : "hover:scale-105 shadow-lg"}
        ${disabled ? "cursor-not-allowed opacity-50" : "cursor-pointer hover:shadow-xl"}
        ${kept ? "ring-4 ring-yellow-400" : ""}
      `}
    >
      <svg viewBox="0 0 100 100" className="w-full h-full">
        <rect
          x="5"
          y="5"
          width="90"
          height="90"
          rx="15"
          fill={kept ? "#ec4899" : "#f472b6"}
          stroke={kept ? "#db2777" : "#ec4899"}
          strokeWidth="3"
        />
        {!rolling && getDotPositions(value)}
      </svg>
      {kept && (
        <div className="absolute -top-2 -right-2 w-6 h-6 bg-yellow-400 rounded-full flex items-center justify-center text-xs font-bold">
          🔒
        </div>
      )}
    </button>
  )
}

export default Dice
