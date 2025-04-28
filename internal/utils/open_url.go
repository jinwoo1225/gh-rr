package utils

import (
	"fmt"
	"os/exec"
)

// OpenURL opens the given URL in the default browser.
func OpenURL(u string) {
	if cmd, err := exec.LookPath("xdg-open"); err == nil {
		exec.Command(cmd, u).Start()
	} else if cmd, err := exec.LookPath("open"); err == nil {
		exec.Command(cmd, u).Start()
	} else {
		fmt.Printf("Please open this URL manually: %s\n", u)
	}
}
