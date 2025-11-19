// src/pages/HomePage.tsx

import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import * as api from '../services/api'; 
import { showToast } from '../services/toast'; 

const HomePage: React.FC = () => {
    const navigate = useNavigate();
    
    const [username, setUsername] = useState('');
    const [joinCode, setJoinCode] = useState('');
    const [isLoading, setIsLoading] = useState(false);

    // Validierung (Akzeptanzkriterium: Username 3-20 Zeichen)
    const usernameLength = username.trim().length;
    const isUsernameValid = usernameLength >= 3 && usernameLength <= 20;

    /**
     * Allgemeine Funktion zur Abwicklung der Guest-Erstellung und Weiterleitung
     * (Verwendet Promise<void> für Typsicherheit)
     */
    const handleAction = async (
        actionCallback: () => Promise<void>, 
        code: string
    ) => {
        if (!isUsernameValid) {
            showToast('Fehler: Benutzername muss 3 bis 20 Zeichen lang sein.', 'error');
            return;
        }

        setIsLoading(true); // Loading-State starten
        
        try {
            await api.createGuest(username.trim()); // API-Call: createGuest
            await actionCallback(); 
            
            // Redirect nach Lobby-Erstellung/Beitritt
            navigate(`/lobby/${code.toUpperCase()}`); 

        } catch (error) {
            // Error-Handling: Toast/Alert bei Fehler
            const errorMessage = error instanceof Error ? error.message : 'Ein unbekannter Fehler ist aufgetreten.';
            showToast(errorMessage, 'error');
            
        } finally {
            setIsLoading(false); 
        }
    };

    /**
     * Handler für "Neue Lobby erstellen"
     */
    const handleCreateLobby = () => {
        const createAndGetCode = async () => {
            // API-Call: createLobby
            await api.createLobby();
        };
        handleAction(createAndGetCode, 'ABCD'); 
    };

    /**
     * Handler für "Lobby beitreten"
     */
    const handleJoinLobby = () => {
        if (joinCode.trim().length !== 4) {
            showToast('Fehler: Join-Code muss 4 Zeichen lang sein.', 'error');
            return;
        }
        
        const join = async () => {
            // API-Call: joinLobby
            await api.joinLobby(joinCode.trim());
        };
        handleAction(join, joinCode.trim()); 
    };

    const LoadingSpinner: React.FC = () => (
        <span className="loading-spinner">Lade...</span>
    );

    return (
        <div className="container" style={{ padding: '20px', maxWidth: '400px', margin: '0 auto', textAlign: 'center' }}>
            
            <h1>Knuffel Startseite</h1>
            
            {/* Username Input-Feld */}
            <div style={{ marginBottom: '20px' }}>
                <label htmlFor="username">Dein Username (3-20 Zeichen):</label>
                <input
                    id="username"
                    type="text"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    maxLength={20}
                    placeholder="Name"
                    disabled={isLoading}
                    style={{ border: !isUsernameValid && usernameLength > 0 ? '1px solid red' : '1px solid gray' }}
                />
                {/* Validation Feedback */}
                {!isUsernameValid && usernameLength > 0 && (
                    <p style={{ color: 'red', fontSize: '12px' }}>Mindestens 3 Zeichen erforderlich.</p>
                )}
            </div>

            {/* Neue Lobby erstellen Button */}
            <div style={{ marginBottom: '20px' }}>
                <button
                    onClick={handleCreateLobby}
                    disabled={isLoading || !isUsernameValid}
                >
                    {isLoading ? <LoadingSpinner /> : 'Neue Lobby erstellen'}
                </button>
            </div>

            <p style={{ margin: '20px 0' }}>--- ODER ---</p>

            {/* Lobby beitreten (Input + Button) */}
            <div style={{ display: 'flex', gap: '10px' }}>
                <input
                    type="text"
                    value={joinCode}
                    onChange={(e) => setJoinCode(e.target.value.toUpperCase().slice(0, 4))}
                    maxLength={4}
                    placeholder="CODE"
                    disabled={isLoading}
                    style={{ flexGrow: 1, textAlign: 'center' }}
                />
                <button
                    onClick={handleJoinLobby}
                    disabled={isLoading || !isUsernameValid || joinCode.length !== 4}
                >
                    Beitreten
                </button>
            </div>
        </div>
    );
}

export default HomePage;