// src/services/toast.ts

/**
 * Zeigt eine Toast/Alert Nachricht an. 
 * Wird später durch eine echte Toast-Komponente ersetzt.
 */
export const showToast = (message: string, type: 'error' | 'success') => {
    console.log(`[TOAST ${type.toUpperCase()}] ${message}`);
    // Fürs Erste nutzen wir die Browser-Alert-Box, um den Fehler zu sehen.
    alert(message); 
};