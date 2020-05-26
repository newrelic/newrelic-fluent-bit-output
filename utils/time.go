package utils

// May 18th 2033 in seconds (Jan 24th 1970 if understood as milliseconds)
const maxSeconds = 2000000000
// May 18th 2033 in millis (Jan 24th 1970 if understood as microseconds)
const maxMilliseconds = maxSeconds * 1000
// May 18th 2033 in microseconds (Jan 24th 1970 if understood as nanoseconds)
const maxMicroseconds = maxMilliseconds * 1000

// TimeToMillis transforms an integer with arbitrary units into milliseconds, by
// doing the following assumptions, in this specific order:
//
//   * If the number is smaller than the number of milliseconds representing the
//     date Jan 24th 1970, then the units are assumed to be [seconds]
//   * If the number is smaller than the number of microseconds representing the
//     date Jan 24th 1970, then the units are assumed to be [milliseconds]
//   * If the number is smaller than the number of nanosecondds representing the
//     date Jan 24th 1970, then the units are assumed to be [microseconds]
//   * Otherwise, the units are assumed to be [nanoseconds]
//
// WARNING: when the current date becomes May 18th 2033, the function below will
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
