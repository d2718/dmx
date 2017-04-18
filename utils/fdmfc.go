// fdmfc.go
//
// a FatDMenu File Chooser implementation in Go
// see https://github.com/d2718/dmx/
//
// by Dan Hill
// last update: 2017-04-17
//
package main

import( "flag"; "fmt"; "os"; "path/filepath"; "sort"; "strings"
        "github.com/d2718/dmx"
)

const DEBUG bool = false

var caseSensitiveSort bool = false          // set by cmd-line flag
var pathSeparator rune = '/'                // set by OS in init()
var hiddenIndicator byte = byte('.')        // *nix-specific?
var directorySelector *DirEntry             // created in init()
var hiddenShower, hiddenHider *DirEntry     // created in init()

func dbglog(fmtstr string, args ...interface{}) {
    if DEBUG {
        fmt.Fprintf(os.Stderr, fmtstr, args...)
    }
}

func die(err error, msgfmt string, args ...interface{}) {
    if msgfmt != "" {
        fmt.Fprintf(os.Stderr, msgfmt, args...)
    }
    if DEBUG && err != nil {
        panic(err)
    } else {
        os.Exit(1)
    }
}

// Type DirEntry is used to represent an entry in a directory listing.
//
type DirEntry struct {
    name  string
    isDir bool
    sort  string
}

// NewDirEntry() creates a new DirEntry struct from an instance of
// os.FileInfo (the type returned by os.File.Readdir()).
//
func NewDirEntry(fi os.FileInfo) *DirEntry {
    de := DirEntry{
        name: fi.Name(),
        isDir: fi.IsDir(),
    }
    if caseSensitiveSort {
        de.sort = de.name
    } else {
        de.sort = strings.ToLower(de.name)
    }
    return &de
}

// DirEntry implements dmx.Item.
//
func (de DirEntry) Key() string { return "" }

func (de DirEntry) MenuLine(_ int) []byte {
    if de.isDir {
        return []byte(fmt.Sprintf("%s%c", de.name, pathSeparator))
    } else {
        return []byte(de.name)
    }
}

func (de DirEntry) SortsBefore(itm dmx.Item) bool {
    rhs := itm.(*DirEntry)
    if de.isDir == rhs.isDir {
        if de.sort < rhs.sort {
            return true
        } else {
            return false
        }
    } else {
        return de.isDir
    }
}

func (de DirEntry) isHidden() bool {
    return de.name[0] == hiddenIndicator        // hacky
}

func isHidden(fi os.FileInfo) bool {
    return fi.Name()[0] == hiddenIndicator      // similarly hacky
}

// parsePath() splits a path into a heirarchical series of names.
//
// Example:
//
// parsePath("/home/dan/.config") returns
// []string{"/", "home", "dan", ".config"}
//
func parsePath(pathname string) []string {
    dbglog("parsePath(%v) called\n", pathname)
    d, f := filepath.Dir(pathname), filepath.Base(pathname)
    dbglog("dir: %v, file: %v\n", d, f)
    
    if f == string(pathSeparator) {
        return []string{f}
    } else if d == string(pathSeparator) {
        return []string{d, f}
    } else {
        return append(parsePath(d), f)
    }
}

// selectPath() is the meat. It repeatedly runs dmenu so the user can
// navigate through the filesystem and select a path.
//
// For proper function, the string slice argument should be the output of
// parsePath() called on an actual directory.
//
func selectPath(pathElts []string, returnDir bool, showHidden bool) string {
    var cur_path string
    for {
        cur_path = filepath.Join(pathElts...)
        df, err := os.Open(cur_path)
        if err != nil {
            die(err, "Error opening directory %#v.\n", cur_path)
        }
        actual_filez, err := df.Readdir(0)
        if err != nil {
            die(err, "Error reading directory %#v.\n", cur_path)
        }
        df.Close()
        
        entriez := make(dmx.ItemList, 0, len(actual_filez)+2)
        if returnDir {
            entriez = append(entriez, directorySelector)
        }
        if showHidden {
            entriez = append(entriez, hiddenHider)
        } else {
            entriez = append(entriez, hiddenShower)
        }
        for _, fi := range actual_filez {
            if showHidden || !isHidden(fi) {
                entriez = append(entriez, NewDirEntry(fi))
            }
        }
        if returnDir {
            sort.Sort(entriez[2:])
        } else {
            sort.Sort(entriez[1:])
        }
        
        dmx_output, err := dmx.DmenuSelect(cur_path, entriez)
        if err != nil {
            dbglog("Error in dmx.DmenuSelect(): %s\n", err)
        }
        
        if dmx_output == nil {
            pel := len(pathElts)
            if pel <= 1 {
                return ""
            } else {
                pathElts = pathElts[:pel-1]
                continue
            }
        }
        
        choice := dmx_output.(*DirEntry)
        if choice == directorySelector {
            return cur_path
        } else if choice == hiddenShower {
            showHidden = true
        } else if choice == hiddenHider {
            showHidden = false
        } else if choice.isDir {
            pathElts = append(pathElts, choice.name)
        } else {
            return filepath.Join(cur_path, choice.name)
        }
    }
}
        
func init() {
    pathSeparator     = os.PathSeparator
    
    directorySelector = &DirEntry{
                            name: fmt.Sprintf("%c [ select current directory ]",
                                              os.PathSeparator),
                            sort: string([]byte{byte(1)}),
                            isDir: false,
                        }
    hiddenShower = &DirEntry{
                        name: fmt.Sprintf("%c [ show hidden files ]",
                                          hiddenIndicator),
                        sort: string([]byte{byte(2)}),
                        isDir: false,
                    }
    hiddenHider = &DirEntry{
                        name: fmt.Sprintf("%c [ hide hidden files ]",
                                          hiddenIndicator),
                        sort: string([]byte{byte(2)}),
                        isDir: false,
                    }
}

func main() {
    var selectDirectory bool = false
    var showHidden bool = false
    var outputFormat string = "%s\n"
    var altCfg string = ""
    var err error = nil
    
    flag.BoolVar(&selectDirectory,   "d", false, "allow Directory selection")
    flag.BoolVar(&showHidden,        "h", false, "show Hidden files by default")
    flag.BoolVar(&caseSensitiveSort, "s", false, "case-Sensitive filename sorting")
    flag.StringVar(&outputFormat,    "f", "%s\n", "output Formatting string")
    flag.StringVar(&altCfg,          "config", "", "specify alternate CONFIGuration file")
    flag.Parse()
    if altCfg == "" {
        dmx.Autoconfigure(nil)
    } else {
        dmx.Autoconfigure([]string{altCfg})
    }
    dbglog("directory arg: %v\n", flag.Arg(0))
    baseDir := flag.Arg(0)
    if baseDir == "" {
        baseDir, err = filepath.Abs(".")
    } else {
        baseDir, err = filepath.Abs(baseDir)
    }
    if err != nil {
        die(err, "Unable to parse provided path: %#v.\n", baseDir)
    }
    dbglog("baseDir: %v\n", baseDir)
    
    v := selectPath(parsePath(baseDir), selectDirectory, showHidden)
    
    fmt.Printf(outputFormat, v)
}
