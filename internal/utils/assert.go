package utils

func Assert(cond bool) {
	if cond {
		return
	}
	panic("")
}

func AssertMsg(cond bool, v ...any) {
	if cond {
		return
	}
	LogError(v)
	panic("")
}

func AssertFail(v ...any) {
	LogError(v)
	panic("")
}
