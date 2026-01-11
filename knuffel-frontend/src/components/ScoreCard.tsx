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
  const upperFields = ["ones", "twos", "threes", "fours", "fives", "sixes"]
  const upperSection = scores.filter((s) => upperFields.includes(s.name))
  const lowerSection = scores.filter((s) => !upperFields.includes(s.name))

  const upperSum = upperSection.reduce((sum, s) => sum + (s.score || 0), 0)
  const hasBonus = upperSum >= 63
  const bonus = hasBonus ? 35 : 0
  const totalScore = scores.reduce((sum, s) => sum + (s.score || 0), 0) + bonus

  const renderField = (field: ScoreField) => (
    <div
      key={field.name}
      className={`
        flex justify-between items-center p-3 rounded-lg border-2
        ${field.score !== null ? "bg-pink-100 border-pink-300 shadow-sm" : "bg-white border-pink-200"}
        ${field.available && !disabled ? "hover:bg-pink-50 cursor-pointer hover:border-pink-400 hover:scale-[1.02]" : ""}
        ${disabled || !field.available ? "cursor-not-allowed opacity-60" : ""}
        transition-all duration-200
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
    <div className="bg-white/90 backdrop-blur-md rounded-3xl p-6 border-4 border-pink-300 shadow-2xl">
      <h2 className="text-2xl font-black text-pink-600 mb-6 text-center uppercase tracking-wider">Punkteblock</h2>

      {/* Upper Section */}
      <div className="mb-6">
        <div className="flex justify-between items-center mb-2 px-1">
          <h3 className="text-xs font-black text-pink-400 uppercase tracking-widest">Oberer Teil</h3>
          <span className="text-xs font-bold text-gray-400">{upperSum} / 63</span>
        </div>
        <div className="grid grid-cols-1 gap-2">{upperSection.map(renderField)}</div>

        {/* Bonus Row */}
        <div className={`
          flex justify-between items-center p-3 mt-2 rounded-lg border-2 border-dashed
          ${hasBonus ? "bg-yellow-50 border-yellow-300 text-yellow-700" : "bg-gray-50 border-gray-200 text-gray-400"}
        `}>
          <span className="font-bold">Bonus (ab 63 Pkt.)</span>
          <span className="font-bold">{hasBonus ? "+35" : "0"}</span>
        </div>
      </div>

      {/* Lower Section */}
      <div>
        <h3 className="text-xs font-black text-pink-400 mb-2 uppercase tracking-widest px-1">Unterer Teil</h3>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-1 gap-2">
          {lowerSection.map(renderField)}
        </div>
      </div>

      {/* Total */}
      <div className="mt-8 pt-6 border-t-4 border-pink-300">
        <div className="flex justify-between items-center p-4 bg-gradient-to-r from-pink-500 via-rose-500 to-pink-600 rounded-2xl text-white shadow-xl transform hover:scale-[1.02] transition-all">
          <div className="flex flex-col">
            <span className="text-xs font-black opacity-80 uppercase tracking-widest">Gesamtpunktzahl</span>
            <span className="font-bold text-lg">Total Score</span>
          </div>
          <span className="font-black text-4xl drop-shadow-md">{totalScore}</span>
        </div>
      </div>
    </div>
  )
}

export default ScoreCard
