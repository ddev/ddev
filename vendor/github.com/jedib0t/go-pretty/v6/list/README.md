## List
[![Go Reference](https://pkg.go.dev/badge/github.com/jedib0t/go-pretty/v6/list.svg)](https://pkg.go.dev/github.com/jedib0t/go-pretty/v6/list)

Pretty-print lists with multiple levels/indents into ASCII/Unicode strings.

  - Append Items one-by-one or as a group
  - Indent/UnIndent as you like
  - Support Items with Multiple-lines
  - Mirror output to an io.Writer object (like os.StdOut)
  - Completely customizable styles
    - Many ready-to-use styles: [style.go](style.go)
  - Render as:
    - (ASCII/Unicode) List
    - HTML List (with custom CSS Class)
    - Markdown List

```
 ■ Game Of Thrones
   ■ Winter
   ■ Is
   ■ Coming
     ■ This
     ■ Is
     ■ Known
 ■ The Dark Tower
   ■ The Gunslinger
```

A demonstration of all the capabilities can be found here:
[../cmd/demo-list](../cmd/demo-list)
