package runv

type initiables []Initable

func (s initiables) Len() int           { return len(s) }
func (s initiables) Less(i, j int) bool { return orderof(s[i], StateInit) < orderof(s[j], StateInit) }
func (s initiables) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type startups []Startup

func (s startups) Len() int { return len(s) }
func (s startups) Less(i, j int) bool {
	return orderof(s[i], StateStartup) < orderof(s[j], StateStartup)
}
func (s startups) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type shutdowns []Shutdown

func (s shutdowns) Len() int { return len(s) }
func (s shutdowns) Less(i, j int) bool {
	return orderof(s[i], StateShutdown) < orderof(s[j], StateShutdown)
}
func (s shutdowns) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type servables []Servable

func (s servables) Len() int           { return len(s) }
func (s servables) Less(i, j int) bool { return orderof(s[i], StateServe) < orderof(s[j], StateServe) }
func (s servables) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func orderof(v interface{}, state State) int {
	if o, ok := v.(Liveorder); ok {
		return o.Order(state)
	}
	return 0
}
