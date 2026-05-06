package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GenerateID(prefix string, existingIDs []string) string {
	idSet := make(map[string]bool)
	for _, id := range existingIDs {
		idSet[id] = true
	}

	makeID := func(hexLen int) string {
		bytes := make([]byte, (hexLen+1)/2)
		_, _ = rand.Read(bytes)
		return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(bytes)[:hexLen])
	}

	for attempts := 0; attempts < 100; attempts++ {
		id := makeID(4)
		if !idSet[id] {
			return id
		}
	}

	for {
		id := makeID(8)
		if !idSet[id] {
			return id
		}
	}
}
