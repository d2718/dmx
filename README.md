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
func (c Choice) Key() string {
    return c.Token
}

func (c Choice) MenuLine(width int) []byte {
    ml_str := fmt.Sprintf("%-*s    %s\n", width, c.Token, c.Description)
    return []byte(ml_str)
}

// We don't care about sorting these, but we need to implement dmx.Item.
func (c Choice) SortsBefore(itm dmx.Item) bool {
    return true
}
```

