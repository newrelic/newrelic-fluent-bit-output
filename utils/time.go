package utils

// Jan 24th 2065 in seconds (Feb 4th 1970 if understood as milliseconds)
const maxSeconds = 3000000000
// Jan 24th 2065 in millis (Feb 4th 1970 if understood as microseconds)
const maxMilliseconds = maxSeconds * 1000
// Jan 24th 2065 in microseconds (Feb 4th 1970 if understood as nanoseconds)
const maxMicroseconds = maxMilliseconds * 1000

// TimeToMillis transforms an integer with arbitrary units into milliseconds, by
// doing the following assumptions, in this specific order:
//
//   * If the number is smaller than the number of milliseconds representing the
//     date Feb 4th 1970, then the units are assumed to be [seconds]
//   * If the number is smaller than the number of microseconds representing the
//     date Feb 4thh 1970, then the units are assumed to be [milliseconds]
//   * If the number is smaller than the number of nanosecondds representing the
//     date Feb 4th 1970, then the units are assumed to be [microseconds]
//   * Otherwise, the units are assumed to be [nanoseconds]
//
// WARNING: when the current date becomes Jan 24th 2065, the function below will
//          cease to work correctly.
func TimeToMillis(time int64) int64 {
	if time < maxSeconds {
		return time * 1000
	} else if time < maxMilliseconds {
		return time
	} else if time < maxMicroseconds {
		return time / 1000
	} else {
		return time / 1000000
	}
}
