package main

import (
	//"context"
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	//"google.golang.org/genai"
)

func main() {
	var ignore []string = readIgnoreFile(".aicomignore")

	args := []string{"diff", "--cached", "--", "."}
	args = append(args, ignore...)

	diffCmd := exec.Command("git", args...)
	diffOut, err := diffCmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Git diff output:\n%s\n", string(diffOut))
}

func readIgnoreFile(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Error opening file: %v", err)
	}

	if file == nil {
		return nil
	}

	scanner := bufio.NewScanner(file)
	var ignores []string

	for scanner.Scan() {

		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		formatedLine := fmt.Sprintf(":(exclude)%s", line)
		ignores = append(ignores, formatedLine)
	}
	defer file.Close()

	return ignores
}
