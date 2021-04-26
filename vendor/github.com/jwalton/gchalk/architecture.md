# Architecture

## Chalk

GChalk's architecture is heavily inspired by Chalk's architecture.

First of all, in Chalk when you dereference `chalk.blue`, what's actually happening here is that you're calling into a `getter` for `blue`.  There are [default getters defined](https://github.com/chalk/chalk/blob/9bf298571eeee20001ba9ff5158b07d2d8a67ec1/source/index.js#L61-L67) through prototype inheritance which don't actually produce a color - instead they redefine `.blue` on the instance you're calling into to create a new "styler" and a new "builder".

The styler for `.blue` has an `open`, `close`, `openAll`, and `closeAll`.  The styler for `.blue.bgGreen` has the styler for `.blue` as a parent, and has an `.open`, `.close` for bgGreen and a `.openAll`, `.closeAll` for both bgGreen and blue.  Chalk is quite clever; since the default getters redefine `.blue` and `.blue.bgGreen` as you call them, these computed stylers are saved for future invocations, so they don't need to be rebuilt.

When we run a styler (in [`applyStyle()`](https://github.com/chalk/chalk/blob/9bf298571eeee20001ba9ff5158b07d2d8a67ec1/source/index.js#L159)), if the string has no escape codes inside, we just append the `openAll` and `closeAll` of the leaf styler, and we're done.

If there are escape codes, then we need to worry about reapplying our styles - if we're coloring something green, and there's a blue bit in the middle, we still need to add the `openAll` and `closeAll` to the start and end, but then we also need to replace every blue "close tag" with a green "open tag".  There's a different close tag for each kind of open tag (fg, bg, and one for each kind of modifier).  To do this, we start at the leaf and replace all the closeTag instances with openTag, then we walk to the parent and do the same, all the way up the chain.

## GChalk

GChalk operates on more or less the same principal.  When you call `WithBlue()`, it will return a Builder instance.  Calling `WithBlue().BgGreen(...)` creates a Blue builder (and associated `stylerData`), and then a BgGreen builder.  Just like in Chalk, the stylerData in the BgGreen builder will have Blue's stylerData as `.parent`.  We of course can't overwrite functions on a receiver in Golang, but instead we store a pointer to each generated child builder in the parent builder; when you call `.WithBlue()` on the root builder, it will create a new builder and store it in `.blue` on the root builder.  This mimics Chalk's behavior, with excellent performance.  (At one point I tried returning a `Builder` interface from `WithBlue()` and the other `With...()` functions, but surprisingly this made coloring a string about four times slower.)

This does mean that calling `.WithBlue()` will immediately allocate a structure with 41 pointers in it.  If you call every permutation of `bgcolor.color.modifier` we end up allocating `16 * 16 * 9 * 41 = 94464` pointers, or about 750K of RAM (although... no one is going to do this).  Calling the `RGB()` and other color model functions don't do this (we don't allocate 16.7m pointers) but they do still create an object which will need to be garbage collected eventually.  If you're doing something like color gradients, you're better off calling directly into `pkg/ansistyles` to color things.

One possible change here would be; instead of storing 41 pointers, store child colors in a `childBuilders map[string]*Builder`.  So then when you call `WithBlue()` we'd go find the builder in `builder.childBuilders["blue"]` instead of in `builder.blue`.  This is very slightly slower at runtime (around 70ns/op vs 60ns/op on my MacBook) but would save some RAM in excessive use cases.  One other thing here is that we could use a trick like this to cache RGB() and other color model functions; we'd obviously want to set a cap on how many we cache (we, again, do not want to cache 16.7m objects).  This could improve performance quite a bit for cases where someone is using a small number of hex/RGB colors, though.