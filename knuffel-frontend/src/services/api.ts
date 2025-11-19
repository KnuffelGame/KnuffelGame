// src/services/api.ts

const throwMockError = (message: string) => {
    throw new Error(message);
};

export const createGuest = async (username: string) => {
    console.log(`[API MOCK] Gast erstellen: ${username}`);
    if (username.toLowerCase().includes('fail')) {
        throwMockError('Gast-Erstellung fehlgeschlagen (Mock-Fehler).');
    }
    return { success: true }; 
};

export const createLobby = async () => {
    console.log("[API MOCK] Lobby erstellen...");
    return { 
        data: { 
            code: 'ABCD', // Der Join-Code der neuen Lobby
            lobbyId: 'mock-lobby-123'
        } 
    };
};

export const joinLobby = async (code: string) => {
    console.log(`[API MOCK] Lobby beitreten: ${code}`);
    if (code.toUpperCase() !== 'ABCD') {
        throwMockError('Lobby-Code ist ungültig.');
    }
    return { success: true };
};