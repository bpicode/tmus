// Package tabs provides a domain-independent tab bar component for Bubble Tea.
// Parent models own the tab content and route messages to content models.
// View renders the tab bar; Render encloses supplied content in a shared frame.
// Parents set the component's size and use ContentSize to size content models.
// Clients supply single-line labels and enough width to display all tabs.
// New accepts WithKeyMap and WithStyles options to override its defaults.
//
// Models start unfocused. Call Focus to enable keyboard navigation, assign the
// model returned by Update, and return its command to Bubble Tea. Selection is
// updated synchronously; the command reports the transition through ChangeMsg.
// Programmatic selection does not emit a command.
package tabs
