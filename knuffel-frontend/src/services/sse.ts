/**
 * SSE (Server-Sent Events) Service for real-time updates
 */

export type SSEEventType =
  | "player_joined"
  | "player_left"
  | "player_kicked"
  | "leader_changed"
  | "game_started"
  | "game_updated"

export interface SSEEvent {
  type: SSEEventType
  data: any
}

/**
 * Connect to SSE stream for lobby updates
 * @param lobbyId - Lobby ID
 * @param onEvent - Event handler callback
 */
export const connectToLobbySSE = (lobbyId: string, onEvent: (event: SSEEvent) => void): EventSource => {
  // TODO: Replace with actual SSE endpoint
  const SSE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080"
  const eventSource = new EventSource(`${SSE_URL}/sse/lobby/${lobbyId}`, {
    withCredentials: true,
  })

  eventSource.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      onEvent(data)
    } catch (error) {
      console.error("Error parsing SSE event:", error)
    }
  }

  eventSource.onerror = (error) => {
    console.error("SSE connection error:", error)
    // TODO: Add reconnection logic
  }

  return eventSource
}

/**
 * Connect to SSE stream for game updates
 * @param gameId - Game ID
 * @param onEvent - Event handler callback
 */
export const connectToGameSSE = (gameId: string, onEvent: (event: SSEEvent) => void): EventSource => {
  // TODO: Replace with actual SSE endpoint
  const SSE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080"
  const eventSource = new EventSource(`${SSE_URL}/sse/game/${gameId}`, {
    withCredentials: true,
  })

  eventSource.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      onEvent(data)
    } catch (error) {
      console.error("Error parsing SSE event:", error)
    }
  }

  eventSource.onerror = (error) => {
    console.error("SSE connection error:", error)
  }

  return eventSource
}
