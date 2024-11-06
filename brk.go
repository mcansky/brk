package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	checkGitRepo()
	if len(os.Args) < 2 {
		displayHelp()
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "update":
		update(args)
	case "refresh":
		refresh(args)
	case "rehydrate":
		rehydrate(args)
	case "split":
		split(args)
	case "push":
		push(args)
	case "mv":
		renameBranch(args)
	case "cleanup":
		cleanup()
	case "hide":
		stash()
	case "pack":
		pack()
	case "recent":
	    recent()
	case "shove":
		shove()
	case "status":
		status()
	case "cherry-log":
	    if len(args) < 1 {
	        fmt.Println("Usage: brk cherry-log <branch>")
	        return
	    }
	    cherryLog(args[0])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		displayHelp()
	}
}

func displayHelp() {
	fmt.Println(`Usage: brk <command> [options]

Commands:
  update      Jump to branch, fetch origin, and rebase
  refresh     Merge branch into the current branch
  rehydrate   Rebase on a specific branch
  split       Create a new branch
  push        Push to remote
  mv          Rename branch
  cleanup     Cleanup branches older than 1 month
  cherry-log  List status of the branch commits against master
  hide        Stash current changes
  pack        Propose changes to commit
  recent	  Show 5 most recent branches
  shove       Amend changes to the previous commit
  status      List changes in color and propose a git diff in bat

Use "brk <command> --help" for more information on a specific command.`)
}

func checkGitRepo() {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		fmt.Println("Error: Not a git repository. Please run this command inside a git repository.")
		os.Exit(1)
	}
}

func execute(command string) {
	fmt.Printf("Executing: %s\n", command)
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing command: %s\n", err)
	}
}

func currentBranch() string {
	output, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		fmt.Printf("Error getting current branch: %s\n", err)
		return ""
	}
	return strings.TrimSpace(string(output))
}

func recent() {
	// Get the 5 most recent branches sorted by commit date
	cmd := exec.Command("sh", "-c", "git branch --sort=-committerdate --format='%(refname:short)' | head -n 5")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error fetching recent branches: %s\n", err)
		return
	}

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(branches) == 0 {
		fmt.Println("No recent branches found.")
		return
	}

	fmt.Println("Recent branches:")
	for i, branch := range branches {
		fmt.Printf("[%d] %s\n", i+1, branch)
	}

	fmt.Print("Enter the number of the branch to switch to (or press Enter to exit): ")
	var choice string
	fmt.Scanln(&choice)

	if choice == "" {
		fmt.Println("No branch selected. Exiting.")
		return
	}

	selectedIndex := -1
	fmt.Sscanf(choice, "%d", &selectedIndex)
	if selectedIndex < 1 || selectedIndex > len(branches) {
		fmt.Println("Invalid selection. Exiting.")
		return
	}

	selectedBranch := branches[selectedIndex-1]
	execute(fmt.Sprintf("git checkout %s", selectedBranch))
}

func update(args []string) {
	branch := flag.NewFlagSet("update", flag.ExitOnError)
	branchName := branch.String("branch", "master", "Branch to update (default: master)")
	remoteName := branch.String("remote", "origin", "Remote to fetch from (default: origin)")

	branch.Parse(args)

	execute(fmt.Sprintf("git checkout %s", *branchName))
	execute(fmt.Sprintf("git fetch %s", *remoteName))
	execute(fmt.Sprintf("git rebase %s/%s", *remoteName, *branchName))
}

func refresh(args []string) {
	branch := flag.NewFlagSet("refresh", flag.ExitOnError)
	branchName := branch.String("branch", "master", "Branch to merge from (default: master)")

	branch.Parse(args)

	execute(fmt.Sprintf("git merge %s", *branchName))
}

func rehydrate(args []string) {
	branch := flag.NewFlagSet("rehydrate", flag.ExitOnError)
	branchName := branch.String("branch", "master", "Branch to rebase onto (default: master)")

	branch.Parse(args)

	execute(fmt.Sprintf("git rebase %s", *branchName))
}

func cherryLog(branch string) {
	checkGitRepo() // Ensure we're in a Git repository

	// Default to comparing with master if no branch is provided
	baseBranch := "master"
	if branch == "" {
		fmt.Println("Usage: brk cherry-log <branch>")
		return
	}

	// Run `git cherry` to get commits unique to the branch
	cmd := exec.Command("git", "cherry", baseBranch, branch)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error running git cherry: %s\n", err)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		fmt.Printf("No unique commits found in branch '%s'.\n", branch)
		return
	}

	fmt.Printf("Commits unique to '%s' compared to '%s':\n", branch, baseBranch)
	for _, line := range lines {
		// Parse the output of `git cherry`
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		sign := parts[0]
		commitHash := parts[1]

		// Run `git log --oneline` for each commit hash
		logCmd := exec.Command("git", "log", "--oneline", "-n", "1", commitHash)
		logOutput, logErr := logCmd.Output()
		if logErr != nil {
			fmt.Printf("Error fetching log for commit %s: %s\n", commitHash, logErr)
			continue
		}

		logMessage := strings.TrimSpace(string(logOutput))
		if sign == "+" {
			fmt.Printf("[!master] %s\n", logMessage) // Commit unique to the branch
		} else if sign == "-" {
			fmt.Printf("\033[90m[master] %s\033[0m\n", logMessage) // Commit already in master
		}
	}
}

func split(args []string) {
	split := flag.NewFlagSet("split", flag.ExitOnError)
	branchName := split.String("name", "", "Name of the new branch")

	split.Parse(args)

	if *branchName == "" {
		fmt.Println("Error: Branch name is required.")
		split.Usage()
		return
	}

	execute(fmt.Sprintf("git checkout -b %s", *branchName))
}

func push(args []string) {
	push := flag.NewFlagSet("push", flag.ExitOnError)
	branchName := push.String("branch", "", "Branch to push")
	remoteName := push.String("remote", "origin", "Remote to push to (default: origin)")

	push.Parse(args)

	if *branchName == "" {
		*branchName = currentBranch()
	}

	execute(fmt.Sprintf("git push %s %s", *remoteName, *branchName))
}

func renameBranch(args []string) {
	mv := flag.NewFlagSet("mv", flag.ExitOnError)
	oldName := mv.String("name", "", "Current branch name (defaults to the current branch)")
	newName := mv.String("new-name", "", "New branch name")

	mv.Parse(args)

	if *newName == "" {
		if mv.NArg() == 1 { // If only one argument is provided, assume it's the new name
			*newName = mv.Arg(0)
			*oldName = currentBranch()
		} else {
			fmt.Println("Error: New branch name is required.")
			mv.Usage()
			return
		}
	}

	if *oldName == "" {
		*oldName = currentBranch()
	}

	if *oldName == "" || *newName == "" {
		fmt.Println("Error: Both old and new branch names are required.")
		mv.Usage()
		return
	}

	execute(fmt.Sprintf("git branch -m %s %s", *oldName, *newName))
}

func cleanup() {
	output, err := exec.Command("git", "branch", "--format", "%(refname:short) %(committerdate:relative)").Output()
	if err != nil {
		fmt.Printf("Error getting branch list: %s\n", err)
		return
	}

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, branchInfo := range branches {
		parts := strings.Fields(branchInfo)
		if len(parts) < 2 {
			continue
		}
		branch := parts[0]
		age := parts[1]

		if strings.Contains(age, "month") || strings.Contains(age, "year") {
			fmt.Printf("Branch: %s, Age: %s. Delete? [y/N] ", branch, age)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) == "y" {
				execute(fmt.Sprintf("git branch -d %s", branch))
			}
		}
	}
}

func stash() {
	execute("git stash")
}

func pack() {
	execute("git add -p")
	fmt.Print("Proceed with commit? [Y/n]: ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) == "n" {
		return
	}
	execute("git commit --verbose")
}

func shove() {
	execute("git add -p")
	execute("git commit --amend --no-edit")
}

func status() {
	execute("git status --short --branch")
	fmt.Print("\nProceed with diff? [Y/n]: ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) == "y" || response == "" {
		execute("git diff | bat --paging=always")
	}
}
