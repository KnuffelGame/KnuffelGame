// src/App.tsx

import React from 'react';
import { 
  BrowserRouter as Router, 
  Routes, 
  Route 
} from 'react-router-dom';

// Importieren Sie die implementierten Seiten
import HomePage from './pages/HomePage'; 
import LobbyPage from './pages/LobbyPage'; 

const App: React.FC = () => {
  return (
    // Der Router umgibt die gesamte Anwendung
    <Router>
      <Routes>
        {/* Route für die Startseite (Task 7.6) */}
        <Route path="/" element={<HomePage />} />
        
        {/* Route für die Lobby-Ansicht (Task 7.7) - Das Ziel des Redirects */}
        <Route path="/lobby/:code" element={<LobbyPage />} /> 

        {/* Weitere Routen (z.B. /game/:id) kommen später hierher */}
      </Routes>
    </Router>
  );
};

export default App;