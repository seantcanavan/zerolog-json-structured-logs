package slutil

import (
	"math/rand"
	"time"
)

// StaticNow generate a new, random Now every execution. This helps test more permutations of dates and edge cases
var StaticNow = func() time.Time {
	nowRand := rand.New(rand.NewSource(time.Now().Unix()))
	// Generating a random year, month, day, etc.
	year := nowRand.Intn(2023-2000) + 2000 // random year between 2000 and 2023
	month := time.Month(nowRand.Intn(12) + 1)
	day := nowRand.Intn(28) + 1 // to avoid issues with February, keep it up to 28
	hour := nowRand.Intn(24)
	minute := nowRand.Intn(60)
	second := nowRand.Intn(60)

	// Constructing the random date using the time.Date function
	randomDate := time.Date(year, month, day, hour, minute, second, 0, time.UTC)

	return randomDate
}()

func StaticNowFunc() time.Time {
	return StaticNow
}
