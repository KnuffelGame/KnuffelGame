import type React from "react"

interface BunnyMascotProps {
  className?: string
  size?: "sm" | "md" | "lg"
}

const BunnyMascot: React.FC<BunnyMascotProps> = ({ className = "", size = "md" }) => {
  const sizeClasses = {
    sm: "w-16 h-16",
    md: "w-24 h-24",
    lg: "w-32 h-32",
  }

  return (
    <div className={`${sizeClasses[size]} ${className}`}>
      <svg
  viewBox="0 0 500 500"
  xmlns="http://www.w3.org/2000/svg"
  stroke="#000"
  strokeWidth="6"
  strokeLinecap="round"
  strokeLinejoin="round"
  fill="none"
>
  {/* Ears */}
  <ellipse cx="160" cy="120" rx="70" ry="150" fill="#e75489" />
  <ellipse cx="340" cy="120" rx="70" ry="150" fill="#e75489" />
  <ellipse cx="160" cy="150" rx="40" ry="110" fill="#ffd8e6" stroke="none" />
  <ellipse cx="340" cy="150" rx="40" ry="110" fill="#ffd8e6" stroke="none" />

  {/* Head */}
  <circle cx="250" cy="270" r="200" fill="#e75489" />

  {/* White face area */}
  <ellipse cx="250" cy="320" rx="160" ry="150" fill="#ffffff" />

  {/* Eyes */}
  <circle cx="185" cy="255" r="55" fill="#000" />
  <circle cx="315" cy="255" r="55" fill="#000" />
  <circle cx="205" cy="235" r="25" fill="#fff" stroke="none" />
  <circle cx="335" cy="235" r="25" fill="#fff" stroke="none" />

  {/* Cheeks */}
  <circle cx="170" cy="330" r="35" fill="#f9a8d4" stroke="none" />
  <circle cx="330" cy="330" r="35" fill="#f9a8d4" stroke="none" />

  {/* Nose */}
  <ellipse cx="250" cy="300" rx="20" ry="15" fill="#e75489" />

  {/* Mouth – small, cute */}
  <path d="M235 315 Q250 330 265 315" stroke="#000" strokeWidth="6" fill="none" />

  {/* Teeth – small & cute */}
  <rect
    x="235"
    y="330"
    width="30"
    height="22"
    rx="4"
    fill="#fff"
    stroke="#000"
    strokeWidth="4"
  />
  <line x1="250" y1="330" x2="250" y2="352" stroke="#000" strokeWidth="4" />

  {/* Whiskers */}
  <line x1="120" y1="315" x2="55" y2="300" />
  <line x1="120" y1="330" x2="55" y2="330" />
  <line x1="120" y1="345" x2="55" y2="360" />

  <line x1="380" y1="315" x2="445" y2="300" />
  <line x1="380" y1="330" x2="445" y2="330" />
  <line x1="380" y1="345" x2="445" y2="360" />
</svg>

    </div>
  )
}

export default BunnyMascot
