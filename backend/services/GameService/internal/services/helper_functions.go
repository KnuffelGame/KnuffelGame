package services

import (
	"errors"
	"log"
	"sort"

	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/models"
)

// validateTurn prüft, ob der User laut TurnOrder an der Reihe ist
func (s *GameService) validateTurn(game *models.GameDB, userID string) error {
	turnOrder := game.TurnOrder

	// Sicherheitscheck: Leere TurnOrder
	if len(turnOrder) == 0 {
		log.Println("CRITICAL: 'turn_order' ist leer bei Validierung")
		return models.ErrInternal
	}

	playerIndex := (game.CurrentTurn - 1) % len(turnOrder)
	expectedPlayerID := turnOrder[playerIndex]

	if expectedPlayerID.PlayerID != userID {
		log.Printf("DEBUG: Not your turn. expectedPlayerID=%s, userID=%s\n\n", expectedPlayerID, userID)
		return ErrNotYourTurn
	}

	return nil
}

func CalculateFieldScore(dice []int, fieldName string) (int, error) {
	if len(dice) != 5 {
		return 0, errors.New("es müssen genau 5 Würfel sein")
	}

	// 1. Vorberechnung: Summe und Häufigkeiten
	sum := 0
	counts := make(map[int]int) // Wie oft kommt jede Zahl vor?
	for _, val := range dice {
		if val < 1 || val > 6 {
			return 0, errors.New("ungültiger Würfelwert")
		}
		sum += val
		counts[val]++
	}

	switch fieldName {
	// --- OBERER BLOCK ---
	case "ones":
		return counts[1] * 1, nil
	case "twos":
		return counts[2] * 2, nil
	case "threes":
		return counts[3] * 3, nil
	case "fours":
		return counts[4] * 4, nil
	case "fives":
		return counts[5] * 5, nil
	case "sixes":
		return counts[6] * 6, nil
	case "bonus":
		return 0, models.ErrBonusCannotBeSelected
	// --- UNTERER BLOCK ---
	case "three_of_a_kind":
		// Mindestens 3 gleiche
		for _, count := range counts {
			if count >= 3 {
				return sum, nil
			}
		}
		return 0, nil

	case "four_of_a_kind":
		// Mindestens 4 gleiche
		for _, count := range counts {
			if count >= 4 {
				return sum, nil
			}
		}
		return 0, nil

	case "full_house":
		// 3 einer Zahl und 2 einer anderen (oder 5 gleiche gilt oft auch als Full House)
		hasThree := false
		hasTwo := false
		for _, count := range counts {
			if count == 3 {
				hasThree = true
			}
			if count == 2 {
				hasTwo = true
			}
			if count == 5 {
				// Ein "Kniffel" kann oft als Full House (Joker) eingetragen werden -> 25 Punkte
				return 25, nil
			}
		}
		if hasThree && hasTwo {
			return 25, nil
		}
		return 0, nil

	case "small_straight":
		// 4 aufeinanderfolgende Zahlen (1-2-3-4, 2-3-4-5, 3-4-5-6)
		if maxSequence(dice) >= 4 {
			return 30, nil
		}
		return 0, nil

	case "large_straight":
		// 5 aufeinanderfolgende Zahlen
		if maxSequence(dice) >= 5 {
			return 40, nil
		}
		return 0, nil

	case "kniffel":
		// Alle 5 gleich
		for _, count := range counts {
			if count == 5 {
				return 50, nil
			}
		}
		return 0, nil

	case "chance":
		return sum, nil

	default:
		return 0, errors.New("ungültiges Feld: " + fieldName)
	}
}

// maxSequence ermittelt die Länge der längsten aufeinanderfolgenden Reihe
func maxSequence(dice []int) int {
	// Duplikate entfernen und sortieren für die Reihen-Berechnung
	uniqueMap := make(map[int]bool)
	var uniqueDice []int
	for _, d := range dice {
		if !uniqueMap[d] {
			uniqueMap[d] = true
			uniqueDice = append(uniqueDice, d)
		}
	}
	sort.Ints(uniqueDice)

	maxSeq := 0
	currentSeq := 0

	for i := 0; i < len(uniqueDice); i++ {
		if i == 0 {
			currentSeq = 1
		} else {
			// Ist die aktuelle Zahl genau 1 größer als die vorherige?
			if uniqueDice[i] == uniqueDice[i-1]+1 {
				currentSeq++
			} else {
				currentSeq = 1
			}
		}
		if currentSeq > maxSeq {
			maxSeq = currentSeq
		}
	}
	return maxSeq
}
