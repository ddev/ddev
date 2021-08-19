package list

// RenderMarkdown renders the List in the Markdown format. Example:
//    * Game Of Thrones
//      * Winter
//      * Is
//      * Coming
//        * This
//        * Is
//        * Known
//    * The Dark Tower
//      * The Gunslinger
func (l *List) RenderMarkdown() string {
	// make a copy of the original style and ensure it is restored on exit
	originalStyle := l.style
	defer func() {
		if originalStyle == nil {
			l.style = nil
		} else {
			l.SetStyle(*originalStyle)
		}
	}()

	// override whatever style was set with StyleMarkdown
	l.SetStyle(StyleMarkdown)

	// render like a regular list
	return l.Render()
}
