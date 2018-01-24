package api

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

type ShortHash [7]byte

func (h ShortHash) String() string {
	return string(h[:])
}
