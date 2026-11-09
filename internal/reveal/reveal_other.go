//go:build !windows

package reveal

import "fmt"

func OpenAndSelect(path string) error {
	if err := CheckFile(path); err != nil {
		return err
	}
	return fmt.Errorf("opening Explorer is only available on Windows")
}
