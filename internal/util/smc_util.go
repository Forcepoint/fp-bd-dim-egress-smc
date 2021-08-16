package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func SmcRulesInclude(exportDirPath string, fileNameToInclude string) error {

	ruleIncludePath := filepath.Join(exportDirPath, "rules_include.config")

	f, err := os.OpenFile(ruleIncludePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return errors.Wrap(err, "error opening rules_include.config")
	}

	defer f.Close()

	if _, err := f.WriteString(fmt.Sprintf("include %s\n", fileNameToInclude)); err != nil {
		return errors.Wrap(err, "error writing to rules_include.config")
	}

	return nil
}
