package honeybadger

// A Context represents a Honeybadger context and contains a mapping of keys to
// arbitrary values. The context is serialized into JSON and sent to Honeybadger
// along with reported notices.
type Context map[string]interface{}

// Get gets the value associated with the given key. If there are no values
// associated with the key, Get returns nil.
func (ctx Context) Get(key string) interface{} {
	return ctx[key]
}

// Set sets the context entry associated with key to the supplied value which
// will be serialized into JSON when sending a notice to Honeybadger. It
// replaces any existing value associated with key.
func (ctx Context) Set(key string, value interface{}) {
	ctx[key] = value
}

// Del deletes the value associated with key.
func (ctx Context) Del(key string) {
	delete(ctx, key)
}
