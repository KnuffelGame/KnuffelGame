"use client"

import type React from "react"

interface ScoreField {
  name: string
  displayName: string
  score: number | null
  available: boolean
}

interface ScoreCardProps {
  scores: ScoreField[]
  onSelectField: (fieldName: string) => void
  disabled?: boolean
}

const ScoreCard: React.FC<ScoreCardProps> = ({ scores, onSelectField, disabled = false }) => {
  const upperSection = scores.filter((s) => ["ones", "twos", "threes", "fours", "fives", "sixes"].includes(s.name))

  const lowerSection = scores.filter((s) => !["ones", "twos", "threes", "fours", "fives", "sixes"].includes(s.name))

  const renderField = (field: ScoreField) => (
    <div
      key={field.name}
      className={`
        flex justify-between items-center p-3 rounded-lg border-2
        ${field.score !== null ? "bg-pink-100 border-pink-300" : "bg-white border-pink-200"}
        ${field.available && !disabled ? "hover:bg-pink-50 cursor-pointer hover:border-pink-400" : ""}
        ${disabled || !field.available ? "cursor-not-allowed opacity-60" : ""}
        transition-all
      `}
      onClick={() => field.available && !disabled && onSelectField(field.name)}
    >
      <span className="font-semibold text-gray-700">{field.displayName}</span>
      <span className={`font-bold ${field.score !== null ? "text-pink-600" : "text-gray-400"}`}>
        {field.score !== null ? field.score : "-"}
      </span>
    </div>
  )

  return (
    <div className="bg-white/90 backdrop-blur-sm rounded-2xl p-6 border-4 border-pink-300 shadow-xl">
      <h2 className="text-2xl font-bold text-pink-600 mb-4 text-center">Punkteblock</h2>

      {/* Upper Section */}
      <div className="mb-4">
        <h3 className="text-sm font-bold text-gray-600 mb-2 uppercase">Oberer Teil</h3>
        <div className="space-y-2">{upperSection.map(renderField)}</div>
      </div>

      {/* Lower Section */}
      <div>
        <h3 className="text-sm font-bold text-gray-600 mb-2 uppercase">Unterer Teil</h3>
        <div className="space-y-2">{lowerSection.map(renderField)}</div>
      </div>

      {/* Total */}
      <div className="mt-4 pt-4 border-t-4 border-pink-300">
        <div className="flex justify-between items-center p-3 bg-gradient-to-r from-pink-500 to-rose-500 rounded-lg text-white">
          <span className="font-bold text-lg">Gesamt</span>
          <span className="font-bold text-2xl">{scores.reduce((sum, s) => sum + (s.score || 0), 0)}</span>
        </div>
      </div>
    </div>
  )
}

export default ScoreCard
