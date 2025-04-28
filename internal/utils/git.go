package utils

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/cli/go-gh/v2"
	"github.com/pkg/errors"
)

// GetBaseDir returns the directory to clone PRs into.
func GetBaseDir() string {
	if b := os.Getenv("BASE_DIR"); b != "" {
		return ExpandHome(b)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "workspace")
}

// ExpandHome expands a leading '~' in a path to the user home directory.
func ExpandHome(p string) string {
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return filepath.Join(home, p[1:])
	}
	return p
}

func CloneAndCheckout(ctx context.Context, repositoryNameWithOwner string, prNumber int, baseDir string) {
	log.Println("Cloning or Checking out", repositoryNameWithOwner, "to", baseDir)
	dir := filepath.Join(baseDir, repositoryNameWithOwner)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		reader := bufio.NewReader(os.Stdin)
		log.Printf("Clone %s into %s? [Y/n]: ", repositoryNameWithOwner, dir)
		resp, _ := reader.ReadString('\n')
		resp = strings.TrimSpace(resp)
		if resp != "" && strings.ToLower(resp) == "n" && !strings.Contains(strings.ToLower(resp), "y") {
			fmt.Println("Skipping clone.")
			return
		}

		log.Printf("Creating Directory into %s\n", dir)
		if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
			log.Println(err)
		}

		log.Printf("Cloning %s into %s\n", repositoryNameWithOwner, dir)
		if _, stderr, err := gh.ExecContext(ctx, "repo", "clone", repositoryNameWithOwner, dir); err != nil {
			log.Panicln(stderr.String(), err)
		}
	} else {
		log.Println("Found existing repository", repositoryNameWithOwner)
	}

	log.Printf("Changing working directory to %s\n", dir)
	if err := os.Chdir(dir); err != nil {
		log.Panicln(errors.Wrap(err, "chdir"))
	}

	// Checkout PR
	log.Println("Checking out", repositoryNameWithOwner+"/"+strconv.Itoa(prNumber))
	if _, stderr, err := gh.ExecContext(ctx, "pr", "checkout", strconv.Itoa(prNumber)); err != nil {
		log.Panicln(stderr.String(), err)
	}

	log.Printf("Checked out PR #%d in %s\n", prNumber, dir)
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	// replace current process with a shell in the checked-out repo
	if err := syscall.Exec(shell, []string{shell}, os.Environ()); err != nil {
		log.Panicln(errors.Wrap(err, "exec shell"))
	}
}
