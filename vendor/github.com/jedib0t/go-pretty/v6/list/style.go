package list

import "github.com/jedib0t/go-pretty/v6/text"

// Style declares how to render the List (items).
type Style struct {
	Format           text.Format // formatting for the Text
	CharItemSingle   string      // the bullet for a single-item list
	CharItemTop      string      // the bullet for the top-most item
	CharItemFirst    string      // the bullet for the first item
	CharItemMiddle   string      // the bullet for non-first/non-last item
	CharItemVertical string      // the vertical connector from one bullet to the next
	CharItemBottom   string      // the bullet for the bottom-most item
	CharNewline      string      // new-line character to use
	LinePrefix       string      // prefix for every single line
	Name             string      // name of the Style
}

var (
	// StyleDefault renders a List like below:
	//  * Game Of Thrones
	//    * Winter
	//    * Is
	//    * Coming
	//      * This
	//      * Is
	//      * Known
	//  * The Dark Tower
	//    * The Gunslinger
	StyleDefault = Style{
		Format:           text.FormatDefault,
		CharItemSingle:   "*",
		CharItemTop:      "*",
		CharItemFirst:    "*",
		CharItemMiddle:   "*",
		CharItemVertical: "  ",
		CharItemBottom:   "*",
		CharNewline:      "\n",
		LinePrefix:       "",
		Name:             "StyleDefault",
	}

	// StyleBulletCircle renders a List like below:
	//  ● Game Of Thrones
	//    ● Winter
	//    ● Is
	//    ● Coming
	//      ● This
	//      ● Is
	//      ● Known
	//  ● The Dark Tower
	//    ● The Gunslinger
	StyleBulletCircle = Style{
		Format:           text.FormatDefault,
		CharItemSingle:   "●",
		CharItemTop:      "●",
		CharItemFirst:    "●",
		CharItemMiddle:   "●",
		CharItemVertical: "  ",
		CharItemBottom:   "●",
		CharNewline:      "\n",
		LinePrefix:       "",
		Name:             "StyleBulletCircle",
	}

	// StyleBulletFlower renders a List like below:
	//  ✽ Game Of Thrones
	//    ✽ Winter
	//    ✽ Is
	//    ✽ Coming
	//      ✽ This
	//      ✽ Is
	//      ✽ Known
	//  ✽ The Dark Tower
	//    ✽ The Gunslinger
	StyleBulletFlower = Style{
		Format:           text.FormatDefault,
		CharItemSingle:   "✽",
		CharItemTop:      "✽",
		CharItemFirst:    "✽",
		CharItemMiddle:   "✽",
		CharItemVertical: "  ",
		CharItemBottom:   "✽",
		CharNewline:      "\n",
		LinePrefix:       "",
		Name:             "StyleBulletFlower",
	}

	// StyleBulletSquare renders a List like below:
	//  ■ Game Of Thrones
	//    ■ Winter
	//    ■ Is
	//    ■ Coming
	//      ■ This
	//      ■ Is
	//      ■ Known
	//  ■ The Dark Tower
	//    ■ The Gunslinger
	StyleBulletSquare = Style{
		Format:           text.FormatDefault,
		CharItemSingle:   "■",
		CharItemTop:      "■",
		CharItemFirst:    "■",
		CharItemMiddle:   "■",
		CharItemVertical: "  ",
		CharItemBottom:   "■",
		CharNewline:      "\n",
		LinePrefix:       "",
		Name:             "StyleBulletSquare",
	}

	// StyleBulletStar renders a List like below:
	//  ★ Game Of Thrones
	//    ★ Winter
	//    ★ Is
	//    ★ Coming
	//      ★ This
	//      ★ Is
	//      ★ Known
	//  ★ The Dark Tower
	//    ★ The Gunslinger
	StyleBulletStar = Style{
		Format:           text.FormatDefault,
		CharItemSingle:   "★",
		CharItemTop:      "★",
		CharItemFirst:    "★",
		CharItemMiddle:   "★",
		CharItemVertical: "  ",
		CharItemBottom:   "★",
		CharNewline:      "\n",
		LinePrefix:       "",
		Name:             "StyleBulletStar",
	}

	// StyleBulletTriangle renders a List like below:
	//  ▶ Game Of Thrones
	//    ▶ Winter
	//    ▶ Is
	//    ▶ Coming
	//      ▶ This
	//      ▶ Is
	//      ▶ Known
	//  ▶ The Dark Tower
	//    ▶ The Gunslinger
	StyleBulletTriangle = Style{
		Format:           text.FormatDefault,
		CharItemSingle:   "▶",
		CharItemTop:      "▶",
		CharItemFirst:    "▶",
		CharItemMiddle:   "▶",
		CharItemVertical: "  ",
		CharItemBottom:   "▶",
		CharNewline:      "\n",
		LinePrefix:       "",
		Name:             "StyleBulletTriangle",
	}

	// StyleConnectedBold renders a List like below:
	//  ┏━ Game Of Thrones
	//  ┃  ┣━ Winter
	//  ┃  ┣━ Is
	//  ┃  ┗━ Coming
	//  ┃     ┣━ This
	//  ┃     ┣━ Is
	//  ┃     ┗━ Known
	//  ┗━ The Dark Tower
	//     ┗━ The Gunslinger
	StyleConnectedBold = Style{
		Format:           text.FormatDefault,
		CharItemSingle:   "━━",
		CharItemTop:      "┏━",
		CharItemFirst:    "┣━",
		CharItemMiddle:   "┣━",
		CharItemVertical: "┃  ",
		CharItemBottom:   "┗━",
		CharNewline:      "\n",
		LinePrefix:       "",
		Name:             "StyleConnectedBold",
	}

	// StyleConnectedDouble renders a List like below:
	//  ╔═ Game Of Thrones
	//  ║  ╠═ Winter
	//  ║  ╠═ Is
	//  ║  ╚═ Coming
	//  ║     ╠═ This
	//  ║     ╠═ Is
	//  ║     ╚═ Known
	//  ╚═ The Dark Tower
	//     ╚═ The Gunslinger
	StyleConnectedDouble = Style{
		Format:           text.FormatDefault,
		CharItemSingle:   "══",
		CharItemTop:      "╔═",
		CharItemFirst:    "╠═",
		CharItemMiddle:   "╠═",
		CharItemVertical: "║  ",
		CharItemBottom:   "╚═",
		CharNewline:      "\n",
		LinePrefix:       "",
		Name:             "StyleConnectedDouble",
	}

	// StyleConnectedLight renders a List like below:
	//  ┌─ Game Of Thrones
	//  │  ├─ Winter
	//  │  ├─ Is
	//  │  └─ Coming
	//  │     ├─ This
	//  │     ├─ Is
	//  │     └─ Known
	//  └─ The Dark Tower
	//     └─ The Gunslinger
	StyleConnectedLight = Style{
		Format:           text.FormatDefault,
		CharItemSingle:   "──",
		CharItemTop:      "┌─",
		CharItemFirst:    "├─",
		CharItemMiddle:   "├─",
		CharItemVertical: "│  ",
		CharItemBottom:   "└─",
		CharNewline:      "\n",
		LinePrefix:       "",
		Name:             "StyleConnectedLight",
	}

	// StyleConnectedRounded renders a List like below:
	//  ╭─ Game Of Thrones
	//  │  ├─ Winter
	//  │  ├─ Is
	//  │  ╰─ Coming
	//  │     ├─ This
	//  │     ├─ Is
	//  │     ╰─ Known
	//  ╰─ The Dark Tower
	//     ╰─ The Gunslinger
	StyleConnectedRounded = Style{
		Format:           text.FormatDefault,
		CharItemSingle:   "──",
		CharItemTop:      "╭─",
		CharItemFirst:    "├─",
		CharItemMiddle:   "├─",
		CharItemVertical: "│  ",
		CharItemBottom:   "╰─",
		CharNewline:      "\n",
		LinePrefix:       "",
		Name:             "StyleConnectedRounded",
	}

	// StyleMarkdown renders a List like below:
	//    * Game Of Thrones
	//      * Winter
	//      * Is
	//      * Coming
	//        * This
	//        * Is
	//        * Known
	//    * The Dark Tower
	//      * The Gunslinger
	StyleMarkdown = Style{
		Format:           text.FormatDefault,
		CharItemSingle:   "*",
		CharItemTop:      "*",
		CharItemFirst:    "*",
		CharItemMiddle:   "*",
		CharItemVertical: "  ",
		CharItemBottom:   "*",
		CharNewline:      "<br/>",
		LinePrefix:       "  ",
		Name:             "StyleMarkdown",
	}

	// styleTest renders a List like below:
	//  t Game Of Thrones
	//  |f Winter
	//  |m Is
	//  |b Coming
	//  | f This
	//  | m Is
	//  | b Known
	//  b The Dark Tower
	//   b The Gunslinger
	styleTest = Style{
		Format:           text.FormatDefault,
		CharItemSingle:   "s",
		CharItemTop:      "t",
		CharItemFirst:    "f",
		CharItemMiddle:   "m",
		CharItemVertical: "|",
		CharItemBottom:   "b",
		CharNewline:      "\n",
		LinePrefix:       "",
		Name:             "styleTest",
	}
)
