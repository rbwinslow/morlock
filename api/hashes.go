package api

import (
	"regexp"
	"strings"
	"fmt"
)

var (
	HashRE *regexp.Regexp = regexp.MustCompile(`[[:xdigit:]]{40}`)
)

type Hash [40]byte

func (h Hash) Equals(other ShortHash) bool {
	return h.Short() == other
}

func (h Hash) Short() ShortHash {
	var result ShortHash
	copy(result[:], h[:])
	return result
}

func (h Hash) String() string {
	return string(h[:])
}

func MustBeHash(s string) (result Hash) {
	s = strings.Trim(s, " ")
	if !HashRE.Match([]byte(s)) {
		panic(fmt.Sprintf("MustBeHash wasn't: \"%s\" doesn't look like a git hash", s))
	}
	copy(result[:], []byte(s))
	return
}

type ShortHash [7]byte

func (h ShortHash) String() string {
	return string(h[:])
}
