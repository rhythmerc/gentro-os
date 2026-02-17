package apppaths

import (
	"github.com/adrg/xdg"
	"path/filepath"
)

var GentroStorage = filepath.Join(xdg.DataHome, "gentro")
var ArtCache = filepath.Join(GentroStorage, "art")
