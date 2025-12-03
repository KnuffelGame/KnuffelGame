// import axios, { type AxiosInstance, type AxiosError } from "axios"

// // API Client Configuration
// const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080/api"

// // Create Axios instance with default config
// const apiClient: AxiosInstance = axios.create({
//   baseURL: API_BASE_URL,
//   withCredentials: true, // Enable cookie-based authentication
//   headers: {
//     "Content-Type": "application/json",
//   },
// })

// // Response Interceptor for error handling
// apiClient.interceptors.response.use(
//   (response) => response,
//   (error: AxiosError) => {
//     // Handle 401 Unauthorized - redirect to homepage
//     if (error.response?.status === 401) {
//       window.location.href = "/"
//       // TODO: Add toast notification for session expired
//     }

//     // TODO: Add toast/alert for other errors
//     console.error("API Error:", error.response?.data || error.message)
//     return Promise.reject(error)
//   },
// )

// // API Functions

// /**
//  * Create a guest user
//  * @param username - Username (3-20 characters)
//  */
// export const createGuest = async (username: string) => {
//   // TODO: Replace with actual API endpoint
//   const response = await apiClient.post("/auth/guest", { username })
//   return response.data
// }

// /**
//  * Create a new lobby
//  */
// export const createLobby = async () => {
//   // TODO: Replace with actual API endpoint
//   const response = await apiClient.post("/lobby/create")
//   return response.data
// }

// /**
//  * Join an existing lobby
//  * @param code - Lobby join code
//  */
// export const joinLobby = async (code: string) => {
//   // TODO: Replace with actual API endpoint
//   const response = await apiClient.post("/lobby/join", { code })
//   return response.data
// }

// /**
//  * Get lobby details
//  * @param lobbyId - Lobby ID
//  */
// export const getLobby = async (lobbyId: string) => {
//   // TODO: Replace with actual API endpoint
//   const response = await apiClient.get(`/lobby/${lobbyId}`)
//   return response.data
// }

// /**
//  * Start the game
//  * @param lobbyId - Lobby ID
//  */
// export const startGame = async (lobbyId: string) => {
//   // TODO: Replace with actual API endpoint
//   const response = await apiClient.post(`/lobby/${lobbyId}/start`)
//   return response.data
// }

// /**
//  * Kick a player from lobby
//  * @param lobbyId - Lobby ID
//  * @param playerId - Player ID to kick
//  */
// export const kickPlayer = async (lobbyId: string, playerId: string) => {
//   // TODO: Replace with actual API endpoint
//   const response = await apiClient.post(`/lobby/${lobbyId}/kick`, { playerId })
//   return response.data
// }

// /**
//  * Roll the dice
//  * @param gameId - Game ID
//  */
// export const rollDice = async (gameId: string) => {
//   // TODO: Replace with actual API endpoint
//   const response = await apiClient.post(`/game/${gameId}/roll`)
//   return response.data
// }

// /**
//  * Keep or release a die
//  * @param gameId - Game ID
//  * @param diceIndex - Index of the die (0-4)
//  * @param keep - Whether to keep the die
//  */
// export const keepDice = async (gameId: string, diceIndex: number, keep: boolean) => {
//   // TODO: Replace with actual API endpoint
//   const response = await apiClient.post(`/game/${gameId}/keep`, { diceIndex, keep })
//   return response.data
// }

// /**
//  * Select a scoring field
//  * @param gameId - Game ID
//  * @param fieldName - Name of the field to select
//  */
// export const selectField = async (gameId: string, fieldName: string) => {
//   // TODO: Replace with actual API endpoint
//   const response = await apiClient.post(`/game/${gameId}/select`, { fieldName })
//   return response.data
// }

// /**
//  * Get current game state
//  * @param gameId - Game ID
//  */
// export const getGameState = async (gameId: string) => {
//   // TODO: Replace with actual API endpoint
//   const response = await apiClient.get(`/game/${gameId}/state`)
//   return response.data
// }

// export default apiClient





import axios, { type AxiosInstance, type AxiosError } from "axios"

// =========================================================================
// HINWEIS: TOAST-PLACEHOLDER
// =========================================================================

// Da ein echtes Toast-System fehlt, verwenden wir console.error.
// BITTE DIESE FUNKTION DURCH ECHTEN TOAST-CODE ERSETZEN (Task 7.3)!
const showToastError = (message: string) => {
  console.error("TOAST ERROR (SIMULIERT):", message)
}

// =========================================================================
// API CLIENT KONFIGURATION (Task 7.3)
// =========================================================================

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080/api"

const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  withCredentials: true, // Cookie-based authentication
  headers: {
    "Content-Type": "application/json",
  },
})

// Response Interceptor for error handling (Task 7.3)
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    // 401 Unauthorized Handling
    if (error.response?.status === 401) {
      showToastError("Sitzung abgelaufen. Du wirst zur Startseite weitergeleitet.")
      window.location.href = "/" 
      return Promise.reject(error) 
    }

    // General Error Handling
    const errorMessage = error.response?.data?.message || error.message || "Unbekannter API-Fehler."
    showToastError(`Fehler (${error.response?.status || 'Netzwerk'}): ${errorMessage}`)
    
    return Promise.reject(error) 
  },
)


// =========================================================================
// MOCK DATEN SPEICHERN
// =========================================================================

// Simuliert den Zustand des Spiels, um Aktionen zu speichern
let currentMockGameState: any = {
  gameId: "MOCK-GAME-123",
  turnPlayerId: "mock-user-456",
  currentPlayerName: "MaxMustermann",
  dice: [1, 2, 3, 4, 5],
  kept: [false, false, false, false, false],
  rollsLeft: 3,
  scoreCard: {
    Ones: null, Twos: null, Threes: null, Fours: null, Fives: null, Sixes: null,
    ThreeOfAKind: null, FourOfAKind: null, FullHouse: null, SmallStraight: null, LargeStraight: null, Kniffel: null, Chance: null
  },
  players: [
    { id: "mock-user-456", username: "MaxMustermann", score: 0 },
    { id: "player2", username: "Anna", score: 0 },
  ],
  status: "ROLLING"
}

// =========================================================================
// API Functions (Vollständig gemockt für UI-Tests)
// =========================================================================

export const createGuest = async (username: string) => {
  await new Promise(resolve => setTimeout(resolve, 300))
  return { userId: "mock-user-456" }
}

export const createLobby = async () => {
  await new Promise(resolve => setTimeout(resolve, 500))
  return {
    lobbyId: "A1B2C3", 
    joinCode: "A1B2"
  }
}

export const joinLobby = async (code: string) => {
  await new Promise(resolve => setTimeout(resolve, 500))
  if (code.toUpperCase() === "KNUX") {
      return {
        lobbyId: "KNUX01", 
        joinCode: "KNUX"
      }
  } else {
      return Promise.reject({
          response: { status: 404, data: { message: "Lobby-Code ungültig oder nicht gefunden." } } 
      })
  }
}

export const getLobby = async (lobbyId: string) => {
  await new Promise(resolve => setTimeout(resolve, 700))
  if (lobbyId === "A1B2C3" || lobbyId === "KNUX01") {
    return {
      lobbyId: lobbyId,
      joinCode: lobbyId === "A1B2C3" ? "A1B2" : "KNUX",
      leaderId: "mock-user-456", 
      players: [
        { id: "mock-user-456", username: "MaxMustermann", isLeader: true },
        { id: "player2", username: "Anna", isLeader: false },
      ],
    }
  } else {
    return Promise.reject({
        response: { status: 404, data: { message: "Lobby existiert nicht." } } 
    })
  }
}

export const startGame = async (lobbyId: string) => {
  // Simuliert, dass der Start-Request ankommt. Der Redirect zur Game-Page
  // erfolgt in der LobbyPage über das simulierte SSE-Event 'game_started'.
  await new Promise(resolve => setTimeout(resolve, 500))
  console.log(`MOCK: Spielstart für Lobby ${lobbyId} angefragt. (Nächster Schritt: Redirect zu /game/${currentMockGameState.gameId})`)
}

export const kickPlayer = async (lobbyId: string, playerId: string) => {
  await new Promise(resolve => setTimeout(resolve, 500))
  console.log(`MOCK: Spieler ${playerId} aus Lobby ${lobbyId} gekickt.`)
}

// =========================================================================
// GAME API Funktionen (MOCK)
// =========================================================================

/**
 * Simuliert das Würfeln.
 */
export const rollDice = async (gameId: string) => {
  if (currentMockGameState.rollsLeft <= 0) {
    return Promise.reject({ response: { status: 400, data: { message: "Keine Würfelversuche mehr übrig." } } })
  }

  // Würfelt nur die Würfel, die NICHT kept sind.
  currentMockGameState.dice = currentMockGameState.dice.map((die: number, index: number) => {
    if (currentMockGameState.kept[index]) {
      return die // Behaltener Würfel
    }
    return Math.floor(Math.random() * 6) + 1 // Neu würfeln (1-6)
  })

  currentMockGameState.rollsLeft--
  currentMockGameState.status = "ROLLED"
  await new Promise(resolve => setTimeout(resolve, 300))
  return currentMockGameState
}

/**
 * Simuliert das Behalten oder Freigeben eines Würfels.
 */
export const keepDice = async (gameId: string, diceIndex: number, keep: boolean) => {
  if (diceIndex >= 0 && diceIndex < 5) {
    currentMockGameState.kept[diceIndex] = keep
  }
  await new Promise(resolve => setTimeout(resolve, 100))
  return currentMockGameState
}

/**
 * Simuliert das Auswählen eines Wertungsfelds.
 */
export const selectField = async (gameId: string, fieldName: string) => {
  if (currentMockGameState.rollsLeft === 3) {
      return Promise.reject({ response: { status: 400, data: { message: "Du musst zuerst würfeln!" } } })
  }
  if (currentMockGameState.scoreCard[fieldName] !== null) {
      return Promise.reject({ response: { status: 400, data: { message: "Feld bereits belegt." } } })
  }
  
  // MOCK: Trage einen Dummy-Score ein und beende den Zug
  currentMockGameState.scoreCard[fieldName] = 20
  currentMockGameState.rollsLeft = 3
  currentMockGameState.kept = [false, false, false, false, false]
  currentMockGameState.status = "NEXT_TURN"

  console.log(`MOCK: Feld ${fieldName} ausgewählt. Zug beendet.`)
  await new Promise(resolve => setTimeout(resolve, 500))
  return currentMockGameState
}

/**
 * Gibt den aktuellen gemockten Spielzustand zurück.
 */
export const getGameState = async (gameId: string) => {
  await new Promise(resolve => setTimeout(resolve, 300))
  return currentMockGameState
}

export default apiClient