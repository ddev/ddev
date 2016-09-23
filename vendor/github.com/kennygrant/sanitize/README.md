sanitize
========

Package sanitize provides functions to sanitize html and paths with go (golang).

FUNCTIONS


```go
sanitize.Accents(s string) string
```

Accents replaces a set of accented characters with ascii equivalents.

```go
sanitize.BaseName(s string) string
```

BaseName makes a string safe to use in a file name, producing a sanitized basename replacing . or / with -. Unlike Name no attempt is made to normalise text as a path.

```go
sanitize.HTML(s string) string
```

HTML strips html tags with a very simple parser, replace common entities, and escape < and > in the result. The result is intended to be used as plain text. 

```go
sanitize.HTMLAllowing(s string, args...[]string) (string, error)
```

HTMLAllowing parses html and allow certain tags and attributes from the lists optionally specified by args - args[0] is a list of allowed tags, args[1] is a list of allowed attributes. If either is missing default sets are used. 

```go
sanitize.Name(s string) string
```

Name makes a string safe to use in a file name by first finding the path basename, then replacing non-ascii characters.

```go
sanitize.Path(s string) string
```

Path makes a string safe to use as an url path.

