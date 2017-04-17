// fdmcm.go
//
// a FatDMenu Clipboard Manager in Go
// see https://github.com/d2718/dmx/
//
// by Dan Hill
// updated 2017-04-17
//
// A simple clipboard manager to manage multiple clipboard entries.
// It is intended to be invoked via window manager bindings. For example,
// I use i3, and have the following lines in my config file
//
// bindsym $mod+c exec /usr/local/bin/fdmcm -s
// bindsym $mod+v exec /usr/local/bin/fdmcm -r
//
// to bind Windows-C and Windows-V to "cutting" from the primary X selection
// and "pasting" to the clipboard respectively.
//
// Requires xclip ( https://github.com/astrand/xclip )
//
package main

import( "bytes"; "flag"; "fmt"; "io"; "os"; "os/exec"; "path/filepath"
        "regexp"; "sort"; "strconv"
        "github.com/d2718/dmx" )

const DEBUG bool = false

var(
    xclipPath string = "/usr/bin/xclip"
    clipDir string = "/tmp/fdmcm"
    maxPrevLength int = 512
    numericRe *regexp.Regexp
)

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

// readTrimmingWhitespace() collapses sequences of whitespace down to a single
// space character (rune 32). It is used to create prievews of clipboard
// contents for dmenu entries.
//
// Currently "whitespace" is defined as any code point 32 or below. This is
// not very sophisticated, but simple and workable.
//
func readTrimmingWhitespace(frm *os.File, too *bytes.Buffer, n_max int) {
    bytez := make([]byte, n_max, n_max)
    _, err := frm.Read(bytez)
    if err == io.EOF {
        // io.EOF is fine.
    } else if err != nil {
        die(err, "Error reading from file %#v.\n", frm.Name())
    }
    buff := bytes.NewBuffer(bytez)
    
    in_whitespace := true   // Beginning with this set to true discards any
                            // leading whitespace.
    
    for r, _, err := buff.ReadRune(); err != io.EOF; r, _, err = buff.ReadRune() {
        if err != nil {
            die(err, "Error decoding buffer from file %#v.\n", frm.Name())
        }
        if r > 32 {
            in_whitespace = false
            too.WriteRune(r)
        } else {
            if !in_whitespace {
                too.WriteRune(' ')
                in_whitespace = true
            }
        }
    }
}

// An Entry represents a single clipboard item.
//
// The prev element holds the bytes that will be passed to dmenu to make a
// menu item. It can't be populated until the number of the highest-numbered
// clipboard file is known, so that leading zeros can be prepended to the
// number (for both aesthetic reasons and to get dmenu to work ideally).
//
type Entry struct {
    n int
    path string
    prev []byte
}

// Entry implements dmx.Item.
//
func (e Entry) Key() string {
    return strconv.Itoa(e.n)
}
func (e Entry) MenuLine(k_width int) []byte {
    return []byte(fmt.Sprintf("%0*d    %s\n", k_width, e.n, e.prev))
}
func (e Entry) SortsBefore(itm dmx.Item) bool {
    oi := itm.(*Entry)
    return (e.n > oi.n)
}

// getClips() returns a slice of *Entries (that is, an EntryList) containing
// all currently stored clips.
//
func getClips() dmx.ItemList {
    df, err := os.Open(clipDir)
    if err != nil {
        die(err, "Unable to open clip directory %#v.\n", clipDir)
    }
    filez, err := df.Readdir(0)
    if err != nil {
        die(err, "Unable to read listing of clip directory %#v.\n", clipDir)
    }
    df.Close()
    
    clips := make(dmx.ItemList, 0, len(filez))
    for _, f := range filez {
        if f.Mode() & os.ModeType == 0 {
            if numericRe.FindString(f.Name()) != "" {
                new_n, err := strconv.Atoi(f.Name())
                if err != nil {
                    fmt.Fprintf(os.Stderr, "Error converting filename %#v to integer: %v\n",
                                           f.Name(), err)
                } else {
                    ep := &Entry{
                            n: new_n,
                            path: filepath.Join(clipDir, f.Name()),
                        }
                    buff := new(bytes.Buffer)
                    ef, err := os.Open(ep.path)
                    if err != nil {
                        die(err, "Unable to open file %#v.\n", ep.path)
                    }
                    readTrimmingWhitespace(ef, buff, maxPrevLength)
                    ef.Close()
                    ep.prev = buff.Bytes()

                    clips = append(clips, ep)
                }
            }
        }
    }
    
    sort.Sort(clips)
    return clips
}

// selectClip() runs dmenu externally to select a clipboard file.
//
func selectClip(prompt string) *Entry {
    clips := getClips()
    
    clip, err := dmx.DmenuSelect(prompt, clips)
    if err != nil {
        return nil
    } else {
        return clip.(*Entry)
    }
}

func init() {
    var err error
    
    numericRe, err = regexp.Compile("^\\d+$")
    if err != nil {
        die(err, "Unable to compile regexp.\n")
    }
    
    err = os.MkdirAll(clipDir, 0775)
    if err != nil {
        die(err, "Unable to ensure existence of clip directory %#v.\n", clipDir)
    }
}

func main() {
    var doSave    bool = false
    var doRecall  bool = false
    var doExpunge bool = false
    var doPurge   bool = false
    var altCfg  string = ""
    
    flag.BoolVar(&doSave,    "s", false, "save current PRIMARY selection")
    flag.BoolVar(&doRecall,  "r", false, "recall saved selection to CLIPBOARD")
    flag.BoolVar(&doExpunge, "x", false, "eXpunge a specific clipboard item")
    flag.BoolVar(&doPurge,   "p", false, "remove ALL clipboard items")
    flag.StringVar(&clipDir, "d", "/tmp/fdmcm", "specify an alternate directory for clipboard files")
    flag.StringVar(&altCfg,  "c", "", "specify an alternate configuration file")
    flag.Parse()
    clipDir, err := filepath.Abs(clipDir)
    if err != nil {
        die(err, "Error with specified clipboard directory %#v.\n", clipDir)
    }
    dbglog("clipboard directory: %v\n", clipDir)
    if altCfg == "" {
        dmx.Autoconfigure(nil)
    } else {
        dmx.Autoconfigure([]string{altCfg})
    }
    
    if doSave {
        var next_n int = 0
        clips := getClips()
        if len(clips) > 0 {
            next_n = clips[0].(*Entry).n + 1
        }
        new_path := filepath.Join(clipDir, strconv.Itoa(next_n))
        f_out, err := os.Create(new_path)
        if err != nil {
            die(err, "Unable to create clipboard file %#v.\n", new_path)
        }
        defer f_out.Close()
        err = f_out.Chmod(0644)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Unable to change mode of clipboard file %#v.\n", new_path)
        }
        
        xcmd := exec.Command(xclipPath, "-selection", "primary", "-o")
        xcmd.Stdout = f_out
        err = xcmd.Run()
        if err != nil {
            die(err, "Error executing external process %#v.\n", xclipPath)
        }
    
    } else if doRecall {
        c := selectClip("R>")
        if c == nil {
            os.Exit(0)
        }
        
        xcmd := exec.Command(xclipPath, "-selection", "clipboard", "-i")
        f_in, err := os.Open(c.path)
        if err != nil {
            die(err, "Unable to open clipboard file %#v.\n", c.path)
        }
        xcmd.Stdin = f_in
        err = xcmd.Run()
        if err != nil {
            die(err, "Error running external process %#v.\n", xclipPath)
        }
        
    } else if doExpunge {
        c := selectClip("X>")
        if c == nil {
            os.Exit(0)
        }
        
        err := os.Remove(c.path)
        if err != nil {
            die(err, "Unable to remove clipboard file %#v.\n", c.path)
        }
        
    } else if doPurge {
        var ep *Entry
        clips := getClips()
        for _, c := range clips {
            ep = c.(*Entry)
            err := os.Remove(ep.path)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error removing clipboard file %#v (%v).\n", ep.path, err)
            }
        }
        
    } else {
        fmt.Fprintf(os.Stderr, "fdmcm: mode argument required. Run \"fdmcm -h\" for help.\n")
    }
}
