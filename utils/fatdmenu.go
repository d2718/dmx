// fatdmenu.go
//
// a fatdmenu implementation in Go
// see https://github.com/d2718/dmx/
//
// by Dan Hill
// last update: 2017-04-17
//
package main

import( "encoding/json"; "flag"; "fmt"; "os"; "sort"
        "github.com/d2718/dmx"
)

const DEBUG bool = false

var(
    separator string
    basePrompt string
    dataFileMode os.FileMode = 0664
    catSelector *SpecialEntry
)

func dbglog(fmtstr string, args ...interface{}) {
    if DEBUG {
        fmt.Fprintf(os.Stderr, fmtstr, args...)
    }
}

func die(err error, fmtstr string, args ...interface{}) {
    fmt.Fprintf(os.Stderr, fmtstr, args...)
    if DEBUG && err != nil {
        panic(err)
    } else {
        os.Exit(1)
    }
}

type Entry struct {
    Token string `json:"key"`
    Desc  string `json:"desc"`
    Val   string `json:"val"`
}

type Category struct {
    Token string       `json:"key"`
    Desc  string       `json:"desc"`
    Stuff dmx.ItemList `json:"stuff"`
}

// Entry and Category both implement dmx.Item
//
func (ent Entry) Key() string {
    return ent.Token
}
func (cat Category) Key() string {
    return fmt.Sprintf("%s%s", cat.Token, separator)
}

func (ent Entry) MenuLine(width int) []byte {
    return []byte(fmt.Sprintf("%-*s    %s\n", width, ent.Token, ent.Desc))
}
func (cat Category) MenuLine(width int) []byte {
    return []byte(fmt.Sprintf("%-*s    %s\n", width, cat.Key(), cat.Desc))
}

func (ent Entry) SortsBefore(itm dmx.Item) bool {
    switch x := itm.(type) {
        case *Category:
            return false
        case *Entry:
            return ent.Token < x.Token
        default:    // not meaningful; shouldn't happen
            return false
    }
}

func (cat Category) SortsBefore(itm dmx.Item) bool {
    switch x := itm.(type) {
        case *Category:
            return cat.Token < x.Token
        case *Entry:
            return true
        default:    // not meaningful; shouldn't happen
            return false
    }
}

func (cat *Category) AddItem(itm dmx.Item) {
    cat.Stuff = append(cat.Stuff, itm)
}

// Category.Expunge() removes an Item from somewhere in the Category's
// contents heirarchy.
//
func (cat *Category) Expunge(itm dmx.Item) bool {
    dbglog("%v::Expunge(%v) called\n", cat, itm)
    idx := -1
    for n, i := range cat.Stuff {
        if i == itm {
            idx = n
            break
        } else {
            if x, is_cat := i.(*Category); is_cat {
                found := x.Expunge(itm)
                if found {
                    return true
                }
            }
        }
    }
    
    if idx > -1 {
        cat.Stuff = append(cat.Stuff[:idx], cat.Stuff[idx+1:]...)
        dbglog("cat now looks like: %v\n", *cat)
        return true
    }
    return false
}

// SpecialEntry represents the "choose current directory" option.
// Obviously, it implements dmx.Item.
//
type SpecialEntry struct {
    line []byte
}
func (se SpecialEntry) Key() string { return "" }
func (se SpecialEntry) SortsBefore(itm dmx.Item) bool { return true }
func (se SpecialEntry) MenuLine(_ int) []byte { return se.line }

// InterpretItem() turns an interface{} Unmarshall()ed from a JSON entry
// into the appropriate type of item (recursively, with Categories).
//
func InterpretItem(dat interface{}) (dmx.Item, error) {
    raw := dat.(map[string]interface{})
    
    if v, is_entry := raw["val"]; is_entry {
        e := Entry{
            Token: raw["key"].(string),
            Desc:  raw["desc"].(string),
            Val:   v.(string),
        }
        return &e, nil
    } else {
        nu_stuff := make(dmx.ItemList, 0)
        nu_raw_list := raw["stuff"].([]interface{})
        for _, x := range nu_raw_list {
            nu_itm, err := InterpretItem(x)
            if err != nil {
                die(err, "Unable to interpret item: %v.\n", x)
            }
            nu_stuff = append(nu_stuff, nu_itm)
        }
        c := Category{
            Token: raw["key"].(string),
            Desc:  raw["desc"].(string),
            Stuff: nu_stuff,
        }
        return &c, nil
    }
}

// readFile() parses the nested JSON data in the file indicated by data_file
// and returns a "base Category" containing those items.
//
func readFile(data_file string) (*Category, error) {
    r_val := Category{
        Token: "",
        Desc:  "Base Category",
    }
    i_list := make(dmx.ItemList, 0)
    
    df, err := os.Open(data_file)
    if err != nil {
        die(err, "Error opening data file %#v for read.\n", data_file)
    }
    defer df.Close()
    df_stat, err := df.Stat()
    if err != nil {
        die(err, "Unable to Stat() %#v.\n", data_file)
    }
    dataFileMode = df_stat.Mode()
    dbglog("dataFileMode at read: %v\n", dataFileMode)
    
    dec := json.NewDecoder(df)
    for dec.More() {
        var raw_item interface{}
        err = dec.Decode(&raw_item)
        if err != nil {
            die(err, "Unable to decode item from %#v.\n", data_file)
        }
        var cooked_item dmx.Item
        cooked_item, err = InterpretItem(raw_item)
        if err != nil {
            die(err, "Unable to interpret raw item %v.\n", raw_item)
        }
        i_list = append(i_list, cooked_item)
    }

    r_val.Stuff = i_list
    return &r_val, nil
}

// WriteFile() writes the contents of the provided "base Category" to the
// given path.
//
func writeFile(data_file string, base_cat *Category) {
    of, err := os.OpenFile(data_file, os.O_WRONLY|os.O_TRUNC, dataFileMode)
    if err != nil {
        die(err, "Error opening data file %#v to write.\n", data_file)
    }
    defer of.Close()
    
    for _, itm := range base_cat.Stuff {
        b, err := json.MarshalIndent(itm, "", "  ")
        dbglog("Item %v\nmarshalled:%s\n", itm, b)
        if err != nil {
            die(err, "unable to Marshal() item %v\n", itm)
        }
        //~ err = json.Indent(of, b, "", "  ")
        //~ if err != nil {
            //~ die(err, "Unable to indent output buffer. Is this dumb, or what?\n")
        //~ }
        _, err = of.Write(b)
        if err != nil {
            die(err, "Error writing to data file %#v.\n", data_file)
        }
        _, err = of.WriteString("\n")
        if err != nil {
            die(err, "Error writing to data file %#v.\n", data_file)
        }
    }
}

// heiroSelect() is the meat. It recursively calls dmenu to select an Item
// from the supplied *Category's heirarchy of Items.
//
// It allows for the option to select categories (instead of just entries),
// and to select ONLY categories (as when choosing where to insert a
// new Item).
//
func heiroSelect(cat *Category, prompt string,
                 canSelectCat, onlySelectCat bool) dmx.Item {

    sort.Sort(cat.Stuff)
    list_len := len(cat.Stuff)
    if canSelectCat {
        list_len += 1
    }
    new_list := make(dmx.ItemList, 0, list_len)
    if canSelectCat {
        new_list = append(new_list, catSelector)
    }
    for _, itm := range cat.Stuff {
        switch x := itm.(type) {
            case *Category:
                new_list = append(new_list, x)
            case *Entry:
                if !onlySelectCat {
                    new_list = append(new_list, x)
                }
            default:    // shouldn't happen
                continue
        }
    }
    
    for {
        choice, err := dmx.DmenuSelect(prompt, new_list)
        if err != nil {
            dbglog("Error in dmx.DmenuSelect(): %v\n", err)
            return nil
        }
        
        switch x := choice.(type) {
            case *Entry:
                return x
            case *SpecialEntry:
                if x == catSelector {
                    return cat
                } else {
                    dbglog("DmenuSelect() returns unexpected *SpecialEntry: %v\n", x)
                    return nil
                }
            case *Category:
                new_prompt := prompt + x.Key()
                new_rval := heiroSelect(x, new_prompt, canSelectCat,
                                        onlySelectCat)
                if new_rval != nil {
                    return new_rval
                }
            default:   // shouldn't happen
                dbglog("In heiroSelect(): dmx.DmenuSelect() returns item which falls through switch statement: %v\n", x)
                return nil
        }
    }
}

func main() {
    var addItem bool = false
    var expungeItem bool = false
    var selectCat bool = false
    var appendOutput bool = false
    var outputFormat = "%s\n"
    var outputFile = ""
    var newKey string = ""
    var newDesc string = ""
    var newVal string = ""
    var altCfg string = ""
    
    flag.StringVar(&separator,    "s", "/",    "category separator")
    flag.StringVar(&basePrompt,   "p", "",     "base prompt")
    flag.BoolVar(&addItem,        "n", false,  "add new entry or category")
    flag.BoolVar(&expungeItem,    "x", false,  "eXpunge item")
    flag.BoolVar(&selectCat,      "c", false,  "add or select category instead of entry")
    flag.StringVar(&outputFormat, "f", "%s\n", "output format string (include a %s!)")
    flag.StringVar(&outputFile,   "o", "",     "output file")
    flag.BoolVar(&appendOutput,   "a", false,  "append to output file instead of overwriting")
    flag.StringVar(&newKey,       "k", "",     "new key for added item")
    flag.StringVar(&newDesc,      "d", "",     "new description for added item")
    flag.StringVar(&newVal,       "v", "",     "new output value for added item")
    flag.StringVar(&altCfg,  "config", "",     "specify an alternate configuration file")
    flag.Parse()
    catSelector = &SpecialEntry{
                    line: []byte(fmt.Sprintf("%s [ choose current category ]\n",
                                     separator)),
                }
    if altCfg == "" {
        dmx.Autoconfigure(nil)
    } else {
        dmx.Autoconfigure([]string{altCfg})
    }
    data_file := flag.Arg(0)
    if data_file == "" {
        die(nil, "No data file provided. You must provide a data file.\n")
    }
    
    base_cat_p, err := readFile(data_file)
    if err != nil {
        die(err, "Error reading data file %#v.\n", data_file)
    }

    if addItem {
        if newKey == "" {
            die(nil, "You must specify a key with the -k flag.\n")
        } else if newDesc == "" {
            die(nil, "You must specify a description with the -d flag.\n")
        }
        if !selectCat {
            if newVal == "" {
                die(nil, "You must specify a value with the -v flag.\n")
            }
        }
        
        containerCat := heiroSelect(base_cat_p, basePrompt,
                                    true, true).(*Category)
        if containerCat == nil {
            return
        }
        if selectCat {
            newCat := Category{
                        Token: newKey,
                        Desc:  newDesc,
                        Stuff: make(dmx.ItemList, 0),
                    }
            containerCat.AddItem(&newCat)
        } else {
            newEnt := Entry{
                        Token: newKey,
                        Desc:  newDesc,
                        Val:   newVal,
                    }
            containerCat.AddItem(&newEnt)
        }
        
        writeFile(data_file, base_cat_p)
        
    } else if expungeItem {
        old_itm := heiroSelect(base_cat_p, basePrompt, true, false)
        if old_itm != nil {
            base_cat_p.Expunge(old_itm)
            writeFile(data_file, base_cat_p)
        }
        
    } else {
        uncast_item := heiroSelect(base_cat_p, basePrompt, false, false)
        if uncast_item != nil {
            the_item := uncast_item.(*Entry)
            if outputFile != "" {
                var of_mode os.FileMode = 0664
                var of_flags = os.O_WRONLY | os.O_CREATE
                if appendOutput {
                    of_flags = of_flags | os.O_APPEND
                }
                of_stat, err := os.Stat(outputFile)
                dbglog("of_mode at instantiation: %v\n", of_mode)
                if os.IsNotExist(err) {
                    // Do nothing; file will be created.
                } else if err != nil {
                    die(err, "Problem with output file %#v.\n", outputFile)
                } else {
                    of_mode = of_stat.Mode()
                }
                dbglog("of_mode at file creation: %v\n", of_mode)
                of, err := os.OpenFile(outputFile, of_flags, of_mode)
                if err != nil {
                    die(err, "Problem opening output file %#v.\n", outputFile)
                }
                if !appendOutput {
                    err = of.Truncate(0)
                    if err != nil {
                        die(err, "Error truncating file %#v.\n", outputFile)
                    }
                }
                defer of.Close()
                of.WriteString(fmt.Sprintf(outputFormat, the_item.Val))
                dbglog("output:\n%s", fmt.Sprintf(outputFormat, the_item.Val))
            } else {
                fmt.Printf(outputFormat, the_item.Val)
                dbglog("output:\n%s", fmt.Sprintf(outputFormat, the_item.Val))
            }
        }
    }
}

