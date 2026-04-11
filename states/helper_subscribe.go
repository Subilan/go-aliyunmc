package states

func SubscribeInstanceStatus() (<-chan State[string], func()) {
	store, ok := GetRecordedHubbedStore[string](HSKeyInstanceStatus)
	if !ok {
		ch := make(chan State[string])
		close(ch)
		return ch, func() {}
	}
	return store.Subscribe()
}

func SubscribeServerSnapshot() (<-chan State[ServerStatusState], func()) {
	store, ok := GetRecordedHubbedStore[ServerStatusState](HSKeyServerStatus)
	if !ok {
		ch := make(chan State[ServerStatusState])
		close(ch)
		return ch, func() {}
	}
	return store.Subscribe()
}