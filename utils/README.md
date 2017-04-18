# Utilities centered around the `dmx` package.

### `fatdmenu`

This is a direct descendant of the idea that ultimately led to this repository.
`fatdmenu` reads from a JSON data file, presenting the options to the user
through `dmenu` and allowing them to navigate through an
arbitrarily-deeply-nested menu to select an item. Check out the structure
of `fatdmenu_data.json` and compare it to what you get when you run
```sh
you@system .../dmx/utils$ go run fatdmenu.go fatdmenu_data.json
```
`fatdmenu_data.json` contains data that you might use when employing
`fatdmenu` as a bookmark manager (for, say, `uzbl`). You can, of course,
add new entries and categories by editing the file directly, or via options
on the command line.

### `fdmfc.go` (Fat DMenu File Chooser)

Navigate through your filesystem and pick a path.
```sh
you@system .../dmx/utils$ go run fdmfc.go ~
```
to start in your home directory. Run it with `--help` to see some options.

### `fdmcm` (Fat DMenu Clipboard Manager)

`fdmcm -s` will save any currently-selected text to a file in the clip
directory (by default `/tmp/fdmcm`). `fdmcm -r` will present you with a
`dmenu` list of clip files with previews of their contents; the file you
select will be dumped into the X CLIPBOARD selection, and you can probably
`ctrl-v` it where you want it. I suggest configuring your window manager to
bind these to key commands. Run it with `--help` for all the options.
