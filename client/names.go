package client

import (
	cryptorand "crypto/rand"
	"fmt"
	"math/rand"
	"time"
)

var adjectives = []string{
	"adorable",
	"boopable",
	"cute",
	"crazy",
	"derpy",
	"fluffy",
	"gentle",
	"horny",
	"loyal",
	"loving",
	"squishy",
}

var subjects = []string{
	"fox",
	"vixen",
	"dog",
	"otter",
	"deer",
	"wolf",
	"dragon",
	"raccoon",
	"proto",
	"bunny",
	"yeen",
}

var random = rand.New(rand.NewSource(time.Now().UnixNano()))

func NewName() string {
	adj := adjectives[random.Intn(len(adjectives))]
	subj := subjects[random.Intn(len(subjects))]

	b := make([]byte, 4)
	_, e := cryptorand.Read(b)
	if e != nil {
		panic(e)
	}
	return fmt.Sprintf("%s-%s-%x", adj, subj, b)
}
