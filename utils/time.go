package utils

func TimeToMillis(time int64) int64 {
	// 18 Apr 2019 == 1555612951401 msecs
	const maxSeconds = 2000000000
	const maxMilliseconds = maxSeconds * 1000
	const maxMicroseconds = maxMilliseconds * 1000
	if time < maxSeconds {
		return time * 1000
	} else if time < maxMilliseconds {
		return time
	} else if time < maxMicroseconds {
		return time / 1000
	} else { // Assume nanoseconds
		return time / 1000000
	}
}
