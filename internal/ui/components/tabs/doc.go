// Package tabs provides a domain-independent tab bar component for Bubble Tea.
// Parent models own the tab content and route messages to content models.
// View renders the tab bar; Render encloses supplied content in a shared frame.
// Parents set the component's size and use ContentSize to size content models.
// Clients supply single-line labels and enough width to display all tabs.
// New accepts WithKeyMap and WithStyles options to override its defaults.
package tabs
