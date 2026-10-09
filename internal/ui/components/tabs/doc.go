// Package tabs provides a domain-independent tab bar component for Bubble Tea.
// Parent models own the tab content and route messages to content models.
// View renders the tab bar; Render encloses supplied content in a shared frame.
// Parents set the component's size and use ContentSize to size content models.
// Clients supply single-line labels. Tabs retain their natural widths. When
// they exceed the assigned width, the terminal clips the header and Render
// omits the content's right border and bottom-right corner.
// New accepts WithKeyMap and WithStyles options to override its defaults.
//
// Models start unfocused. Call Focus to enable keyboard navigation and return
// Update's command to Bubble Tea. Update modifies the model synchronously and
// returns the same pointer; assigning that pointer back to the model is optional.
// The command reports the selection transition through ChangeMsg. Programmatic
// selection does not emit a command.
package tabs
