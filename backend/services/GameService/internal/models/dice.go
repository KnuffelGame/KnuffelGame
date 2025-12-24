package models

import "math/rand"

func RollDice(sides int) int {

	return 1 + (rand.Intn(sides))
}
