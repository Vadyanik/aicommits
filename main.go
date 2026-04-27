package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"google.golang.org/genai"
)

func main() {
	yesFlag := flag.Bool("y", false, "automatically apply the commit without prompting")
	printFlag := flag.Bool("p", false, "print the commit message only, without committing")
	apiFlag := flag.String("api", "", "save the API key to config file")
	flag.Parse()

	if *apiFlag != "" {
		saveAPIKey(*apiFlag)
		return
	}

	apiKey := loadAPIKey()
	if apiKey == "" {
		log.Fatal("❌ API key not found. Please set it using: aic -api <your_key>")
	}
	os.Setenv("GOOGLE_API_KEY", apiKey)

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

	if *printFlag {
		fmt.Println(aiMessage)
		return
	}

	shouldApply := *yesFlag
	if !*yesFlag {
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

func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "aicommits", "apikey"), nil
}

func saveAPIKey(key string) {
	configPath, err := getConfigPath()
	if err != nil {
		log.Fatalf("❌ Failed to get config path: %v", err)
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		log.Fatalf("❌ Failed to create config directory: %v", err)
	}

	if err := os.WriteFile(configPath, []byte(key), 0600); err != nil {
		log.Fatalf("❌ Failed to save API key: %v", err)
	}

	fmt.Println("✅ API key saved successfully")
	os.Exit(0)
}

func loadAPIKey() string {
	configPath, err := getConfigPath()
	if err == nil {
		data, err := os.ReadFile(configPath)
		if err == nil && len(data) > 0 {
			return strings.TrimSpace(string(data))
		}
	}

	return os.Getenv("GOOGLE_API_KEY")
}

func askAi(diff []byte, history []byte) string {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	instruction := fmt.Sprintf(
		"Write a highly concise git commit message based on the following diff. "+
			"Output ONLY the message itself, no preamble or quotes. "+
			"STRICT LENGTH LIMIT: Keep it under 50 words total. Do NOT write long paragraphs. "+
			"Use Conventional Commits format (e.g., feat:, fix:, refactor:). "+
			"For simple changes, return ONLY ONE LINE (the header). "+
			"For complex changes, return the header and a maximum of 1-2 short bullet points. "+
			"Try to replicate the style of the last 10 commit messages:\n%s",
		history,
	)	prompt := fmt.Sprintf("%s\n\n%s", instruction, string(diff))

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
