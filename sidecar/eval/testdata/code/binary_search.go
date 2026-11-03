package search

// BinarySearch returns the index of target in the sorted slice, or -1 when it is not there.
func BinarySearch(sorted []int, target int) int {
	lo, hi := 0, len(sorted)-1
	for lo <= hi {
		mid := lo + (hi-lo)/2
		switch {
		case sorted[mid] == target:
			return mid
		case sorted[mid] < target:
			lo = mid + 1
		default:
			hi = mid - 1
		}
	}
	return -1
}
