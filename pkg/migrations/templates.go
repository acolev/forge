package migrations

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
)

func getTemplate(name string) (string, error) {
	tpl, err := ioutil.ReadFile(filepath.Join("stubs", name+".tpl"))
	if err != nil {
		return "", fmt.Errorf("unable to read stub: %v", err)
	}
	return string(tpl), nil
}
