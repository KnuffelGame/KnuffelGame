"use client"

import type React from "react"
import { useEffect, useState } from "react"

// Definiere den Tupel-Typ für eine Koordinate
type Coordinate = [number, number]

// Definiere den Typ für das Positions-Objekt, um den TypeScript-Fehler zu beheben
type PositionsMap = {
  [key in 1 | 2 | 3 | 4 | 5 | 6]: Coordinate[]
}

interface Dice {
  id: number
  x: number
  y: number
  size: number
  rotation: number
  color: string
  animationDuration: number
  animationDelay: number
  value: number
}

const COLORS = ["#ec4899", "#f472b6", "#fbcfe8", "#db2777", "#be185d"]

// Hilfsfunktion: Definiert die Positionen der Punkte für jeden Wert (1-6)
const getDotPositions = (num: number): Coordinate[] => {
  const positions: PositionsMap = {
    1: [[50, 50]],
    2: [[30, 30], [70, 70]],
    3: [[30, 30], [50, 50], [70, 70]],
    4: [[30, 30], [70, 30], [30, 70], [70, 70]],
    5: [[30, 30], [70, 30], [50, 50], [30, 70], [70, 70]],
    6: [[30, 30], [70, 30], [30, 50], [70, 50], [30, 70], [70, 70]],
  }
  return positions[num as keyof PositionsMap] || []
}


const AnimatedDiceBackground: React.FC = () => {
  const [dice, setDice] = useState<Dice[]>([])

  useEffect(() => {
    // Generate random dice
    const generatedDice: Dice[] = Array.from({ length: 30 }, (_, i) => ({ // 1. ANZAHL: 30 Würfel
      id: i,
      x: Math.random() * 100,
      y: Math.random() * 100,
      size: Math.random() * 80 + 10, // 2. GRÖSSE: 10px bis 90px (80 + 10)
      rotation: Math.random() * 360,
      color: COLORS[Math.floor(Math.random() * COLORS.length)],
      animationDuration: Math.random() * 10 + 10,
      animationDelay: Math.random() * 5,
      value: Math.floor(Math.random() * 6) + 1, // Zufälliger Wert (1-6)
    }))
    setDice(generatedDice)
  }, [])

  return (
    <div className="fixed inset-0 overflow-hidden pointer-events-none opacity-20 z-0">
      {dice.map((d) => (
        <div
          key={d.id}
          className="absolute" // "animate-bounce" entfernt (Fix für das Zucken)
          style={{
            left: `${d.x}%`,
            top: `${d.y}%`,
            width: `${d.size}px`,
            height: `${d.size}px`,
            // Nur Rotation beibehalten
            animation: `spin ${d.animationDuration * 1.5}s linear infinite`, 
            animationDelay: `${d.animationDelay}s`,
          }}
        >
          <svg viewBox="0 0 100 100" fill={d.color} style={{ transform: `rotate(${d.rotation}deg)` }}>
            <rect x="10" y="10" width="80" height="80" rx="12" />
            
            {/* Punkte basierend auf dem zufälligen 'value' rendern */}
            {getDotPositions(d.value).map((coord, idx) => (
              <circle key={idx} cx={coord[0]} cy={coord[1]} r="6" fill="white" />
            ))}
          </svg>
        </div>
      ))}
    </div>
  )
}

export default AnimatedDiceBackground