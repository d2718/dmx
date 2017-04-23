// dmx.go
//
// A package for interacting with suckless tools' dmenu.
//
// https://github.com/d2718/dmx
//
// by Dan Hill
// https://github.com/d2718/
// updated 2017-04-22
//
// suckless: http://suckless.org/
// dmenu:    http://tools.suckless.org/dmenu/
//
// This package depends on my dconfig package:
// https://github.com/d2718/dconfig
//
package dmx

import( "bytes"; "fmt"; "os"; "os/exec";
        "github.com/d2718/dconfig"
)

var(
    DmenuPath string = "/usr/local/bin/dmenu"
    Font string =  "-*-fixed-medium-r-normal--13-*-*-*-*-*-ISO10646-*" // "ProggyCleanTTCE-12"
    NormalFG   string = "#000"
    NormalBG   string = "#444"
    SelectedFG string = "#ddd"
    SelectedBG string = "#222"
    CRLF []byte
    crlfLength int
)

// Autoconfigure() reads and responds to a configuration file.
// It first tries all the paths in the supplied slice of strings (which can
// be nil), then in ~/.config/dmx.conf, then in /usr/share/dmx.conf. It stops
// and reads from the first one it finds.
//
func Autoconfigure(other_cfgs []string) error {
    dconfig.Reset()
    dconfig.AddString(&DmenuPath,  "dmenu",       dconfig.STRIP)
    dconfig.AddString(&Font,       "font",        dconfig.STRIP)
    dconfig.AddString(&NormalBG,   "normal_bg",   dconfig.STRIP)
    dconfig.AddString(&NormalFG,   "normal_fg",   dconfig.STRIP)
    dconfig.AddString(&SelectedBG, "selected_bg", dconfig.STRIP)
    dconfig.AddString(&SelectedFG, "selected_fg", dconfig.STRIP)
    
    cfg_files := make([]string, 0, 2)
    for _, fname := range other_cfgs {
        cfg_files = append(cfg_files, fname)
    }
    cfg_files = append(cfg_files, os.ExpandEnv("$HOME/.config/dmx.conf"))
    cfg_files = append(cfg_files, "/usr/share/dmx.conf")
    
    err := dconfig.Configure(cfg_files, false)
    
    return err
}

// Interface Item represents a single menu item to be passed to dmenu.
//
// The intended way for an Item to appear in the menu is:
//
// key     more thorough description of item
//
// Item.MenuLine() Should return a []byte of a similar format. It takes an int
// meant to represent the maximum key length among all the Items displayed by
// dmenu, so that all the descriptions can be lined up properly.
//
// Item.Key() should return the key so that DmenuSelect() can calculate the
// maximum key length of all the items to be displayed.
//
// Item.SortsBefore() allows you to make slices of Item sortable by
// implementing a single method on Item rather than three methods on
// []Item. If you don't need sortability, just implement something trivial
// like
// 
// func (mi MyItem) SortsBefore(itm dmx.Item) bool { return false }
//
type Item interface {
    Key() string
    MenuLine(int) []byte
    SortsBefore(Item) bool
}

// Type ItemList makes it more convenient to implement sorting for your
// particular Item implementor. (See the documentation for Item.)
//
type ItemList []Item

func (il ItemList) Len() int { return len(il) }
func (il ItemList) Swap(i, j int) { il[i], il[j] = il[j], il[i] }
func (il ItemList) Less(i, j int) bool { return il[i].SortsBefore(il[j]) }

func (il ItemList) keyLen() int {
    kl := 0
    var nl int
    for _, itm := range il {
        nl = len(itm.Key())
        if nl > kl {
            kl = nl
        }
    }
    return kl
}

// If the supplied []byte does not end with a newline, it gets appended and
// passed back; otherwise, it just gets passed back.
//
func ensureReturnminated(bs []byte) []byte {
    if len(bs) < crlfLength {
        return append(bs, CRLF...)
    }
    
    if bytes.Equal(bs[len(bs)-crlfLength:], CRLF) {
        return bs
    } else {
        return append(bs, CRLF...)
    }
}

// DmenuSelect() runs dmenu externally to allow the user to select one of the
// Items in the supplied ItemList.
//
func DmenuSelect(prompt string, input ItemList) (Item, error) {
    key_len := input.keyLen()
    menu_lines := make([][]byte, 0, len(input))
    for _, itm := range input {
        menu_lines = append(menu_lines, ensureReturnminated(itm.MenuLine(key_len)))
    }
    
    var dmenu_input  = new(bytes.Buffer)
    var dmenu_output = new(bytes.Buffer)
    for _, ml := range menu_lines {
        dmenu_input.Write(ml)
    }
    
    dcmd := exec.Command(DmenuPath,
                         "-l", fmt.Sprintf("%d", len(input)),
                         "-p",  prompt,     "-fn", Font,
                         "-nb", NormalBG,   "-nf", NormalFG,
                         "-sb", SelectedBG, "-sf", SelectedFG)
    dcmd.Stdin  = dmenu_input
    dcmd.Stdout = dmenu_output
    err := dcmd.Run()
    if err != nil {
        return nil, err
    }
    
    stdout_bytes := dmenu_output.Bytes()
    for n, ml := range menu_lines {
        if bytes.Equal(stdout_bytes, ml) {
            return input[n], nil
        }
    }
    
    return nil, nil
}

// Run() is a more primitive interface to dmenu than DmenuSelect().
// It takes a slice of byte slices (which should NOT be newline-terminated)
// to pass to dmenu and returns the output of dmenu (which WILL be
// newline-terminated).
//
func Run(prompt string, input [][]byte) ([]byte, error) {
    
    dcmd := exec.Command(DmenuPath, "-l", fmt.Sprintf("%d", len(input)),
                                    "-p", prompt,
                                    "-fn", Font,
                                    "-nb", NormalBG, "-nf", NormalFG,
                                    "-sb", SelectedBG, "-sf", SelectedFG)
    
    stdin_slice := make([]byte, 0)
    for _, bs := range input {
        stdin_slice = append(stdin_slice, bs...)
        stdin_slice = append(stdin_slice, CRLF...)
    }
    dcmd.Stdin = bytes.NewReader(stdin_slice)
    var stdout_bytes bytes.Buffer
    dcmd.Stdout = &stdout_bytes
    err := dcmd.Run()
    return stdout_bytes.Bytes(), err
}

func init() {
    CRLF = []byte("\n")
    crlfLength = len(CRLF)
}
