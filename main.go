package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"google.golang.org/genai"
)

func main() {
	// Define -y flag for non-interactive mode
	yesFlag := flag.Bool("y", false, "automatically apply the commit without prompting")
	flag.Parse()

	var ignore []string = readIgnoreFile(".aicomignore")

	args := []string{"diff", "--cached", "--", "."}
	args = append(args, ignore...)

	diffCmd := exec.Command("git", args...)
	diffOut, err := diffCmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	if diffOut == nil || len(diffOut) == 0 {
		fmt.Println("No changes to commit.")
		return
	}

	logCmd := exec.Command("git", "log", "-n", "10", "--format=%s")
	logOut, err := logCmd.Output()
	if err != nil {
		logOut = []byte("")
	}

	var aiMessage string = askAi(diffOut, logOut)

	// Determine if we should apply the commit
	shouldApply := *yesFlag
	if !*yesFlag {
		// Interactive mode: show message and prompt for confirmation
		fmt.Printf("Generated commit message:\n%s\n", aiMessage)
		fmt.Print("Apply this commit? (Y/n): ")
		var answer string
		fmt.Scanln(&answer)
		shouldApply = strings.ToLower(answer) == "y" || strings.ToLower(answer) == "yes" || answer == ""
	}

	if shouldApply {
		if !*yesFlag {
			fmt.Println("Committing changes...")
		}

		commitCmd := exec.Command("git", "commit", "-m", aiMessage)
		output, err := commitCmd.CombinedOutput()
		if err != nil {
			log.Fatalf("Error: commit failed: %v\nOutput: %s", err, string(output))
		}

		if !*yesFlag {
			fmt.Printf("Success!\n%s\n", string(output))
		} else {
			// Quiet mode: only output the commit message
			fmt.Println(aiMessage)
		}
	} else {
		fmt.Println("Commit aborted by user.")
	}
}

func readIgnoreFile(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer file.Close()

	var ignores []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		formatedLine := fmt.Sprintf(":(exclude)%s", line)
		ignores = append(ignores, formatedLine)
	}

	if err := scanner.Err(); err != nil {
		return ignores
	}

	return ignores
}

func askAi(diff []byte, history []byte) string {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	instruction := fmt.Sprintf(
		"Write a concise git commit message based on the following diff. "+
			"Output ONLY the message itself, no preamble or quotes. "+
			"Make them as informative as possible, and try to be creative. "+
			"If the change is tiny (like a shebang fix or typo), use 'fix:' or 'chore:' with a simple description. "+
			"Use Conventional Commits format (e.g., feat:, fix:, docs:). "+
			"Try to replicate the style of the last 10 commit messages:\n%s",
		history,
	)
	prompt := fmt.Sprintf("%s\n\n%s", instruction, string(diff))

	result, err := client.Models.GenerateContent(
		ctx,
		"gemini-2.5-flash-lite",
		genai.Text(prompt),
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}
	return result.Text()
}
