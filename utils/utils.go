package utils

// ZipAndEmumerateTwoArrays ...
func ZipAndEmumerateTwoArrays(first []string, second []string) map[int]Pair {
	enumerated := make(map[int]Pair)
	for i := 0; i < len(second); i++ {
		enumerated[i] = Pair{X: first[i], Y: second[i]}
	}
	return enumerated
}
