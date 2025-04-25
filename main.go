package main

import (
   "bufio"
   "bytes"
   "encoding/json"
   "fmt"
   "io/ioutil"
   "net/http"
   "net/url"
   "os"
   "os/exec"
   "path"
   "path/filepath"
   "sort"
   "strings"
   "time"
)

// IssueItem represents a GitHub issue or pull request returned by the search API.
type IssueItem struct {
   RepositoryURL string    `json:"repository_url"`
   Title         string    `json:"title"`
   HTMLURL       string    `json:"html_url"`
   CreatedAt     time.Time `json:"created_at"`
}

// SearchResult wraps the items field of the GitHub search API response.
type SearchResult struct {
   Items []IssueItem `json:"items"`
}

// Entry is a simplified PR entry for display.
type Entry struct {
   Repo   string
   Title  string
   URL    string
   Age    time.Duration
   AgeStr string
}

func main() {
   // Help flag
   if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
       usage()
       os.Exit(0)
   }
   // Ensure GitHub token
   token := os.Getenv("GITHUB_TOKEN")
   if token == "" {
       fmt.Fprintln(os.Stderr, "Environment variable GITHUB_TOKEN must be set")
       os.Exit(1)
   }
   // Check dependencies
   for _, dep := range []string{"fzf", "git"} {
       if _, err := exec.LookPath(dep); err != nil {
           fmt.Fprintf(os.Stderr, "Error: '%s' is required. Please install %s.\n", dep, dep)
           os.Exit(1)
       }
   }
   baseDir := getBaseDir()
   // Fetch PRs
   revItems, err := fetchIssues(token, "is:pr is:open review-requested:@me draft:false archived:false")
   if err != nil {
       fmt.Fprintf(os.Stderr, "Error fetching review-requested PRs: %v\n", err)
       os.Exit(1)
   }
   myItems, err := fetchIssues(token, "is:pr is:open author:@me archived:false")
   if err != nil {
       fmt.Fprintf(os.Stderr, "Error fetching your PRs: %v\n", err)
       os.Exit(1)
   }
   draftItems, err := fetchIssues(token, "is:pr is:open review-requested:@me draft:true archived:false")
   if err != nil {
       fmt.Fprintf(os.Stderr, "Error fetching draft PRs: %v\n", err)
       os.Exit(1)
   }
   now := time.Now().UTC()
   revEntries := buildEntries(revItems, now)
   myEntries := buildEntries(myItems, now)
   draftEntries := buildEntries(draftItems, now)
   revLines := formatEntries(revEntries)
   myLines := formatEntries(myEntries)
   draftLines := formatEntries(draftEntries)
   if len(revLines) == 0 {
       fmt.Fprintln(os.Stderr, "No review-requested PRs found; press 'm' to view your PRs.")
   }
   // Write temporary files
   revFile, err := writeTempFile("gh-rr-rev", revLines)
   if err != nil {
       fatal(err)
   }
   defer os.Remove(revFile)
   myFile, err := writeTempFile("gh-rr-my", myLines)
   if err != nil {
       fatal(err)
   }
   defer os.Remove(myFile)
   draftFile, err := writeTempFile("gh-rr-draft", draftLines)
   if err != nil {
       fatal(err)
   }
   defer os.Remove(draftFile)
   // Launch fzf
   header := "Select PR: Enter=Open, c=Clone & checkout; m=My PRs, r=Review requests, d=Draft review requests"
   cmd := exec.Command("fzf",
       "--header", header,
       "--delimiter", "\t",
       "--with-nth", "1,2,3",
       "--ansi",
       "--expect", "enter,c",
       "--bind", fmt.Sprintf("m:reload(cat %s)", myFile),
       "--bind", fmt.Sprintf("r:reload(cat %s)", revFile),
       "--bind", fmt.Sprintf("d:reload(cat %s)", draftFile),
   )
   in, err := os.Open(revFile)
   if err != nil {
       fatal(err)
   }
   defer in.Close()
   var out bytes.Buffer
   cmd.Stdin = in
   cmd.Stdout = &out
   cmd.Stderr = os.Stderr
   if err := cmd.Run(); err != nil {
       os.Exit(0)
   }
   lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
   if len(lines) < 2 {
       os.Exit(0)
   }
   key := lines[0]
   selected := lines[1]
   parts := strings.SplitN(selected, "\t", 4)
   if len(parts) < 4 {
       fmt.Fprintln(os.Stderr, "Invalid selection")
       os.Exit(1)
   }
   repo := strings.TrimSpace(parts[0])
   urlStr := strings.TrimSpace(parts[3])
   if key == "c" {
       cloneAndCheckout(repo, urlStr, baseDir, token)
   } else {
       openURL(urlStr)
   }
}

// usage prints the help message.
func usage() {
   fmt.Println(`Usage: gh rr
Open your GitHub review-requested pull requests in a neat fzf picker.

Requires:
  - GITHUB_TOKEN environment variable
  - fzf
  - git

Flags:
  -h, --help    Show this help message and exit.`)
}

// getBaseDir returns the base directory for cloning PRs, checking env and config file.
func getBaseDir() string {
   if b := os.Getenv("BASE_DIR"); b != "" {
       return expandHome(b)
   }
   home, err := os.UserHomeDir()
   if err != nil {
       return ""
   }
   cfg := filepath.Join(home, ".gh-rr")
   if _, err := os.Stat(cfg); err == nil {
       if bash, err := exec.LookPath("bash"); err == nil {
           cmd := exec.Command(bash, "-c", fmt.Sprintf("source %s && printf %%s \"$BASE_DIR\"", cfg))
           out, err := cmd.Output()
           if err == nil {
               s := string(out)
               if s != "" {
                   return expandHome(s)
               }
           }
       }
   }
   return filepath.Join(home, "workspace")
}

// expandHome expands a leading ~ in paths to the user home directory.
func expandHome(pathStr string) string {
   if strings.HasPrefix(pathStr, "~") {
       home, err := os.UserHomeDir()
       if err != nil {
           return pathStr
       }
       return filepath.Join(home, pathStr[1:])
   }
   return pathStr
}

// fetchIssues queries the GitHub API search/issues endpoint.
func fetchIssues(token, query string) ([]IssueItem, error) {
   endpoint := "https://api.github.com/search/issues"
   u := endpoint + "?q=" + url.QueryEscape(query) + "&per_page=100"
   req, err := http.NewRequest("GET", u, nil)
   if err != nil {
       return nil, err
   }
   req.Header.Set("Accept", "application/vnd.github.v3+json")
   req.Header.Set("Authorization", "token "+token)
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
       return nil, err
   }
   defer resp.Body.Close()
   if resp.StatusCode != http.StatusOK {
       body, _ := ioutil.ReadAll(resp.Body)
       return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
   }
   var result SearchResult
   if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
       return nil, err
   }
   return result.Items, nil
}

// buildEntries converts IssueItems to Entries and sorts them by age descending.
func buildEntries(items []IssueItem, now time.Time) []Entry {
   var entries []Entry
   for _, it := range items {
       repo := strings.TrimPrefix(it.RepositoryURL, "https://api.github.com/repos/")
       age := now.Sub(it.CreatedAt)
       ageSec := int(age.Seconds())
       ageStr := humanize(ageSec)
       entries = append(entries, Entry{Repo: repo, Title: it.Title, URL: it.HTMLURL, Age: age, AgeStr: ageStr})
   }
   sort.Slice(entries, func(i, j int) bool {
       return entries[i].Age > entries[j].Age
   })
   return entries
}

// humanize formats seconds into a human-friendly string.
func humanize(s int) string {
   switch {
   case s < 60:
       return fmt.Sprintf("%ds", s)
   case s < 3600:
       return fmt.Sprintf("%dm", s/60)
   case s < 86400:
       return fmt.Sprintf("%dh", s/3600)
   case s < 604800:
       return fmt.Sprintf("%dd", s/86400)
   default:
       return fmt.Sprintf("%dw", s/604800)
   }
}

// formatEntries pads columns and returns lines ready for fzf.
func formatEntries(entries []Entry) []string {
   var maxRepo, maxTitle, maxAge int
   for _, e := range entries {
       if l := len(e.Repo); l > maxRepo {
           maxRepo = l
       }
       if l := len(e.Title); l > maxTitle {
           maxTitle = l
       }
       if l := len(e.AgeStr); l > maxAge {
           maxAge = l
       }
   }
   var lines []string
   for _, e := range entries {
       lines = append(lines, fmt.Sprintf("%-*s\t%-*s\t%-*s\t%s",
           maxRepo, e.Repo, maxTitle, e.Title, maxAge, e.AgeStr, e.URL))
   }
   return lines
}

// writeTempFile writes lines to a temporary file and returns its path.
func writeTempFile(prefix string, lines []string) (string, error) {
   f, err := ioutil.TempFile("", prefix)
   if err != nil {
       return "", err
   }
   defer f.Close()
   for _, line := range lines {
       if _, err := f.WriteString(line + "\n"); err != nil {
           return "", err
       }
   }
   return f.Name(), nil
}

// fatal prints an error and exits.
func fatal(err error) {
   fmt.Fprintf(os.Stderr, "Error: %v\n", err)
   os.Exit(1)
}

// cloneAndCheckout handles cloning and checking out the PR branch.
func cloneAndCheckout(repo, urlStr, baseDir, token string) {
   // Parse PR number from URL
   prNumber := ""
   if u, err := url.Parse(urlStr); err == nil {
       prNumber = path.Base(u.Path)
   } else {
       fatal(fmt.Errorf("invalid URL %s", urlStr))
   }
   cloneDir := filepath.Join(baseDir, repo)
   // Clone if needed
   if info, err := os.Stat(cloneDir); err == nil && info.IsDir() {
       fmt.Printf("Directory '%s' already exists, skipping clone\n", cloneDir)
   } else {
       fmt.Printf("Clone directory: %s\n", cloneDir)
       fmt.Print("Clone into this directory? [Y/n] ")
       reader := bufio.NewReader(os.Stdin)
       line, err := reader.ReadString('\n')
       if err != nil {
           os.Exit(1)
       }
       line = strings.TrimSpace(line)
       if line == "" {
           line = "Y"
       }
       if strings.ToLower(line) != "y" {
           os.Exit(0)
       }
       if err := os.MkdirAll(filepath.Dir(cloneDir), 0755); err != nil {
           fatal(err)
       }
       cmd := exec.Command("git", "clone", fmt.Sprintf("https://github.com/%s.git", repo), cloneDir)
       cmd.Stdout = os.Stdout
       cmd.Stderr = os.Stderr
       if err := cmd.Run(); err != nil {
           fatal(err)
       }
   }
   // Checkout PR branch
   if ghPath, err := exec.LookPath("gh"); err == nil {
       cmd := exec.Command(ghPath, "pr", "checkout", prNumber)
       cmd.Dir = cloneDir
       cmd.Stdout = os.Stdout
       cmd.Stderr = os.Stderr
       if err := cmd.Run(); err != nil {
           fatal(fmt.Errorf("gh pr checkout failed: %v", err))
       }
   } else {
       // Manual fetch & checkout
       prAPI := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%s", repo, prNumber)
       req, err := http.NewRequest("GET", prAPI, nil)
       if err != nil {
           fatal(err)
       }
       req.Header.Set("Accept", "application/vnd.github.v3+json")
       req.Header.Set("Authorization", "token "+token)
       resp, err := http.DefaultClient.Do(req)
       if err != nil {
           fatal(err)
       }
       defer resp.Body.Close()
       if resp.StatusCode != http.StatusOK {
           body, _ := ioutil.ReadAll(resp.Body)
           fatal(fmt.Errorf("HTTP %d: %s", resp.StatusCode, body))
       }
       var pr struct {
           Head struct {
               Ref  string `json:"ref"`
               Repo struct {
                   FullName string `json:"full_name"`
                   CloneURL string `json:"clone_url"`
               } `json:"repo"`
           } `json:"head"`
       }
       if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
           fatal(err)
       }
       headRef := pr.Head.Ref
       headRepoFull := pr.Head.Repo.FullName
       headRepoURL := pr.Head.Repo.CloneURL
       if headRepoFull == repo {
           cmd := exec.Command("git", "fetch", "origin", fmt.Sprintf("pull/%s/head", prNumber))
           cmd.Dir = cloneDir
           cmd.Stdout = os.Stdout
           cmd.Stderr = os.Stderr
           if err := cmd.Run(); err != nil {
               fatal(err)
           }
           cmd = exec.Command("git", "checkout", "-b", headRef, "FETCH_HEAD")
           cmd.Dir = cloneDir
           cmd.Stdout = os.Stdout
           cmd.Stderr = os.Stderr
           if err := cmd.Run(); err != nil {
               fatal(err)
           }
       } else {
           remoteName := fmt.Sprintf("pr-%s", prNumber)
           cmd := exec.Command("git", "remote", "add", remoteName, headRepoURL)
           cmd.Dir = cloneDir
           cmd.Stderr = os.Stderr
           _ = cmd.Run()
           cmd = exec.Command("git", "fetch", remoteName, headRef)
           cmd.Dir = cloneDir
           cmd.Stdout = os.Stdout
           cmd.Stderr = os.Stderr
           if err := cmd.Run(); err != nil {
               fatal(err)
           }
           cmd = exec.Command("git", "checkout", "-b", headRef, "--track", fmt.Sprintf("%s/%s", remoteName, headRef))
           cmd.Dir = cloneDir
           cmd.Stdout = os.Stdout
           cmd.Stderr = os.Stderr
           if err := cmd.Run(); err != nil {
               fatal(err)
           }
       }
   }
   fmt.Printf("Checked out PR #%s in %s\n", prNumber, cloneDir)
   shell := os.Getenv("SHELL")
   if shell == "" {
       shell = "/bin/sh"
   }
   shCmd := exec.Command(shell)
   shCmd.Dir = cloneDir
   shCmd.Stdin = os.Stdin
   shCmd.Stdout = os.Stdout
   shCmd.Stderr = os.Stderr
   shCmd.Run()
}

// openURL opens the URL in the default browser.
func openURL(urlStr string) {
   if xdg, err := exec.LookPath("xdg-open"); err == nil {
       _ = exec.Command(xdg, urlStr).Start()
   } else if opn, err := exec.LookPath("open"); err == nil {
       _ = exec.Command(opn, urlStr).Start()
   } else {
       fmt.Printf("Please open this URL manually: %s\n", urlStr)
   }
}