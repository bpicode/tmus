package tabs

// Option configures a model during construction.
type Option func(*Model)

// WithKeyMap replaces the complete navigation key map.
func WithKeyMap(keys KeyMap) Option {
	return func(m *Model) {
		m.KeyMap = keys
	}
}

// WithStyles replaces the complete style configuration.
func WithStyles(styles Styles) Option {
	return func(m *Model) {
		m.Styles = styles
	}
}
