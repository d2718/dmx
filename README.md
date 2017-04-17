# dmx

A Go package for interacting with `dmenu` and some utilities that use it.

### History

This began many years ago when I was running
[`wmii`](https://code.google.com/archive/p/wmii), which uses
[`dmenu`](http://tools.suckless.org/dmenu/) to present you all the possible
executable options in your `$PATH`. I thought, "Wouldn't it be great if `dmenu`
could present you more descriptive options, yet still spit out the proper
executable path?" And lo&mdash;a Python script to wrap `dmenu` was born.

This evolved through several forms and languages, and has currently alighted
upon Go (and, obviously, its current form).

### Requirements

Obviously, to use this package or anything of the utilities that depend on it,
you must have [`dmenu`](http://tools.suckless.org/dmenu/) installed. (There
may very well be a binary package for your distribution; be aware of the
limitations of your installed version.) `dmx` also relies on my
['dconfig'](https://github.com/d2718/dconfig/) package.

### Overview

`dmenu`'s method of operation is to read a list of options from the standard
input (one option per line), present these options to the user in a way that
makes selecting from them with the keyboard efficient, and then printing the
user's selection on the standard output (followed by a newline). I believe the
original use case was as a sort of program launcher for
[`dwm`](http://dwm.suckless.org/) and `wmii`.

The point of this package is to facilitate selecting more human-readable
options with `dmenu`, but still returning strings that are useful softwarily.
It provides a framework for associating human-readable options with
utility-useful options and using `dmenu` to let the user select them.

### Operation

Each option to be passed to `dmenu` is represented by a struct that implements
the ``dmx.Item`` interface:
```go
type Item interface {
    Key() string
    MenuLine(int) []byte
    SortsBefore(Item) bool
}
```
The most important of these methods is `(dmx.Item) MenuLine(int)`, which
returns the "human-readable" option to be passed to `dmenu`.

This method takes an `int` for aesthetic reasons. To function efficiently with
`dmenu`, it can be helpful for each option to have a "token" or "key"
associated with it. For example, here is a sample set of selections that
might be presented by `dmenu` when running, say, a program launcher:
```
writer    LibreOffice Writer
calc      LibreOffice Calc (Spreadsheet)
uzbl      uzbl Web Browser
term      Sakura Terminal Emulator
wicd      WICD Wireless Client
arandr    X Output Control and Configuration
```
Knowing the length of the longest key that will appear in the menu allows
`MenuLine()` to space the description text appropriately. This is also where
the `(dmx.Item) Key()` method comes into play: It should return a "key" so
that its length can be determined.

A possible implementation that would generate the above menu options might be

```go
// Choice represents a single option in a program launcher.
//
type Choice struct {
    Token string
    Description string
    Value string
}

// Choice implements dmx.Item
func (c Choice) Key() string { return c.Token }

func (c Choice) MenuLine(width int) []byte {
    ml_str := fmt.Sprintf("%-*s    %s\n", width, c.Token, c.Description)
    return []byte(ml_str)
}

// We don't care about sorting these, but we need to implement dmx.Item.
func (c Choice) SortsBefore(itm dmx.Item) bool { return true }
```

We need to pass `dmx.DmenuSelect()` a `dmx.ItemList`, which is just a
slice of `dmx.Item`s. So let's populate our slice.

```go
choices := make(dmx.ItemList, 0, 6)

choices = append(choices, &Choice {
                            Token: "writer",
                            Description: "LibreOffice Writer",
                            Value: "/usr/bin/soffice --nologo --writer",
                          })

// ...
// many lines of appending elided
// ...

choices = append(choices. &Choice {
                            Token: "arandr",
                            Description: "X Output Control and Configuration",
                            Value: "/usr/bin/arandr"
                          })
```

Now passing our slice to `dmx.DmenuSelect()` will cause `dmenu` to present 
the above choices for the user to select, and then return the `dmx.Item`
corresponding to his or her choice:

```go
chosen, err := dmx.DmenuSelect("execute: ", choices)
if err != nil {
    fmt.Sprintf(os.Stderr, "error running dmenu: %s\n", err)
}
```

If `chosen` ends up being non-nil, we can then pass `chosen.Value` to the
shell or `exec.Cmd.Run()` or something.

See the source of the included utilities for complete implementations.

### Included Utilities

The `utils/` directory includes some system utilities that rely on this
library. See the `README` there for more details.

  * `fatdmenu` &mdash The original use case, `fatdmenu` functions similarly
    to the example in the "Overview" section above, but with an
    arbitrarily-deeply-nested heirarchy of menus and submenus. I use this
    program as both a program launcher in conjunction with
    [`i3`](https://i3wm.org/) and as a bookmark manager in conjunction
    with [`uzbl`](https://www.uzbl.org/).
    
  * `fdmcm` ("Fat DMenu Clipboard Manager") &mdash; A clipboard manager for
    storing from and retrieving to X's PRIMARY and CLIPBOARD selections.

  * `fdmfc` ("Fat DMenu File Chooser") &mdash; Use `dmenu` to navigate through
    your machine's filesystem and select a path.
    
