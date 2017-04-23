// dtodo.go
//
// a command-line to-do list
//
// by Dan Hill
//
// updated 2017-04-22
//
package main

import( "bufio"; "flag"; "fmt"; "io"; "os"; "os/exec"; "path/filepath";
        "regexp"; "sort"; "strconv"; "strings"
        "github.com/d2718/dconfig"
        "github.com/d2718/dmx"
)

const DEBUG = false

var(
    editorPath string
    itemsPath string
    listPath string
    altCfg string = ""
    tempDir string = "/tmp"
    formatterPath string = "/usr/bin/pandoc"
    browserPath string = "/usr/bin/uzbl"
    listItemRe *regexp.Regexp
    fileNameRe *regexp.Regexp
)

func rpt(msgfmt string, args ...interface{}) {
    fmt.Fprintf(os.Stderr, msgfmt, args...)
}

func die(err error, msgfmt string, args ...interface{}) {
    fmt.Fprintf(os.Stderr, msgfmt, args...)
    if DEBUG && err != nil {
        panic(err)
    } else {
        os.Exit(1)
    }
}

// Item represents a single item on the to-do list.
//
type Item struct {
    N int
    Title string
}

// Item implements dmx.Item
//
func (itm Item) Key() string { return fmt.Sprintf("%d", itm.N) }
func (itm Item) SortsBefore(oi dmx.Item) bool { return itm.N < oi.(*Item).N }
func (itm Item) MenuLine(w int) []byte {
    return []byte(fmt.Sprintf("%*d  %s\n", w, itm.N, itm.Title))
}

func (itm Item) fileName() string { return fmt.Sprintf("%d.md", itm.N) }

func (itm Item) path() string {
    return filepath.Join(itemsPath, itm.fileName())
}

func (itm Item) print() error {
    inf, err := os.Open(itm.path())
    if err != nil {
        return err
    }
    defer inf.Close()
    _, err = io.Copy(os.Stdout, inf)
    return err
}

func (itm Item) prettyPrint() error {
    tmpfname := filepath.Join(tempDir, fmt.Sprintf("%d.html", os.Getpid()))
    fmtcmd := exec.Command(formatterPath)
    inf, err := os.Open(itm.path())
    if err != nil {
        die(err, "Unable to open item file %v for read: %s\n", itm.path(), err)
    }
    outf, err := os.OpenFile(tmpfname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
    if err != nil {
        inf.Close()
        die(err, "Unable to open temporary file %v for writing: %s\n", tmpfname, err)
    }
    fmtcmd.Stdin = inf
    fmtcmd.Stdout = outf
    err = fmtcmd.Run()
    inf.Close()
    outf.Close()
    if err != nil {
        return err
    }
    viewcmd := exec.Command(browserPath, tmpfname)
    err = viewcmd.Run()
    if err != nil {
        return err
    }
    os.Remove(tmpfname)
    return nil
}

func readList() (dmx.ItemList, error) {
    lf, err := os.Open(listPath)
    if err != nil {
        rpt("Unable to open list file %v: %s\n", listPath, err)
        return nil, nil
    }
    defer lf.Close()
    itemz := make(dmx.ItemList, 0, 0)
    lf_rdr := bufio.NewReader(lf)
    var line []byte
    for line, _, err = lf_rdr.ReadLine(); err == nil; line, _, err = lf_rdr.ReadLine() {
        m := listItemRe.FindSubmatch(line)
        if m != nil {
            n, e := strconv.Atoi(string(m[1]))
            if e != nil {
                rpt("Error converting line \"%s\" to item: %s\n", line, e)
                continue
            }
            itemz = append(itemz, &Item{ N: n, Title: string(m[2]), })
        }
    }
    return itemz, nil
}

func writeList(lst dmx.ItemList) error {
    of, err := os.OpenFile(listPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
    if err != nil {
        return err
    }
    for _, itm := range lst {
        x := itm.(*Item)
        fmt.Fprintf(of, "%d %s\n", x.N, x.Title)
    }
    of.Close()
    return nil
}

func createItem(itm *Item) error {
    new_fn := itm.path()
    of, err := os.OpenFile(new_fn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
    if err != nil {
        return err
    }
    fmt.Fprintf(of, "# %s\n\n", itm.Title)
    of.Close()
    
    if editorPath != "" {
        edcmd := exec.Command(editorPath, new_fn)
        edcmd.Stdin = os.Stdin
        err = edcmd.Run()
        if err != nil {
            return err
        }
    } else {
        of, err = os.OpenFile(new_fn, os.O_WRONLY|os.O_APPEND, 0644)
        if err != nil {
            return err
        }
        _, err := io.Copy(of, os.Stdin)
        of.Close()
        return err
    }
    
    return nil
}

func selectItem(il dmx.ItemList) *Item {
    if altCfg == "" {
        dmx.Autoconfigure(nil)
    } else {
        dmx.Autoconfigure([]string{altCfg})
    }
    itm, err := dmx.DmenuSelect(">", il)
    if err != nil {
        rpt("Error in dmx.DmenuSelect: %v\n", err)
        return nil
    } else if itm == nil {
        return nil
    } else {
        return itm.(*Item)
    }
}

func init() {
    itemsPath = os.ExpandEnv("$HOME/.dtodo")
    editorPath = os.Getenv("EDITOR")
    listItemRe = regexp.MustCompile(`^(\d+)\s+(.+)$`)
    listPath = filepath.Join(itemsPath, "list.txt")
}

func main() {
    var viewFormatted bool = false
    var addDesc     string = ""
    var doExpunge     bool = false
    var doTidy        bool = false
    
    flag.BoolVar(&viewFormatted, "p", false, "view Prettily-formatted output")
    flag.StringVar(&addDesc,     "a", "",    "Add new item")
    flag.BoolVar(&doExpunge,     "x", false, "eXpunge a single item (it's done!)")
    flag.BoolVar(&doTidy,        "t", false, "Tidy the list directory")
    flag.StringVar(&altCfg, "config", "",    "specify alternate CONFIGuration file")
    flag.Parse()
    if addDesc != "" {
        addDesc = strings.Join(append([]string{addDesc}, flag.Args()...), " ")
    }
    config_files := make([]string, 0, 3)
    if altCfg != "" {
        config_files = append(config_files, altCfg)
    }
    config_files = append(config_files, os.ExpandEnv("$HOME/.config/dmx.conf"))
    config_files = append(config_files, "/usr/share/dmx.conf")
    
    dconfig.Reset()
    dconfig.AddString(&itemsPath,     "dtodo_path",      dconfig.STRIP)
    dconfig.AddString(&tempDir,       "dtodo_temp",      dconfig.STRIP)
    dconfig.AddString(&editorPath,    "editor",          dconfig.STRIP)
    dconfig.AddString(&formatterPath, "dtodo_formatter", dconfig.STRIP)
    dconfig.AddString(&browserPath,   "dtodo_viewer",    dconfig.STRIP)
    dconfig.Configure(config_files, false)
    
    lst, err := readList()
    if err != nil {
        die(err, "Error reading list: %v\n", err)
    }
    sort.Sort(lst)
    
    if addDesc != "" {
        n_sup := len(lst) + 1
        for n, itm := range lst {
            x := itm.(*Item)
            if n < x.N {
                n_sup = n
                break
            }
        }
        
        nitm_p := &Item{ N: n_sup, Title: addDesc, }
        err = createItem(nitm_p)
        if err != nil {
            die(err, "Unable to create item %v: %s\n", err)
        }
        lst = append(lst, nitm_p)
        err = writeList(lst)
        if err != nil {
            die(err, "Unable to write to list file: %s\n", err)
        }
        
    } else if doExpunge {
        it := selectItem(lst)
        if it == nil {
            os.Exit(0)
        }
        
        idx := -1
        for n, itm := range lst {
            if it == itm {
                idx = n
                break
            }
        }
        if idx > -1 {
            lst = append(lst[:idx], lst[idx+1:]...)
            if !doTidy {
                err := writeList(lst)
                if err != nil {
                    die(err, "Unable to write list file: %s\n", err)
                }
            }
        }
        
    } else {
        it := selectItem(lst)
        if it != nil {
            if viewFormatted {
                err = it.prettyPrint()
                if err != nil {
                    die(err, "Error prettyPrint()ing item %v: %v\n", it, err)
                }
            } else {
                it.print()
            }
        }
    }
    
    if doTidy {
        fileNameRe := regexp.MustCompile(`^\d+\.md$`)
        used_names := make(map[string]bool)
        for _, itm := range lst {
            used_names[itm.(*Item).fileName()] = true
        }
        idir, err := os.Open(itemsPath)
        if err != nil {
            die(err, "Unable to open items directory %v: %s\n", itemsPath, err)
        }
        fnames, err := idir.Readdirnames(0)
        if err != nil {
            die(err, "Unable to read items directory %v: %s\n", itemsPath, err)
        }
        for _, fname := range fnames {
            if fileNameRe.FindString(fname) != "" {
                if used_names[fname] == false {
                    os.Remove(filepath.Join(itemsPath, fname))
                }
            }
        }
    }
}
