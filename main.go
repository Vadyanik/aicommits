package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/genai"
)

var ollamaHTTPClient = &http.Client{Timeout: 2 * time.Minute}

func main() {
	yesFlag := flag.Bool("y", false, "automatically apply the commit without prompting")
	printFlag := flag.Bool("p", false, "print the commit message only, without committing")
	ollamaFlag := flag.Bool("o", false, "use a local Ollama model instead of Gemini")
	apiFlag := flag.String("api", "", "save the API key to config file")
	flag.Parse()

	if *apiFlag != "" {
		saveAPIKey(*apiFlag)
		return
	}

	apiKey := loadAPIKey()
	if apiKey == "" && !*ollamaFlag {
		log.Fatal("API key not found. Please set it using: aic -api <your_key> or use -o for Ollama")
	}
	if apiKey != "" {
		os.Setenv("GOOGLE_API_KEY", apiKey)
	}

	var ignore []string = readIgnoreFile(".aicomignore")

	args := []string{"diff", "--cached", "--", "."}
	args = append(args, ignore...)

	diffCmd := exec.Command("git", args...)
	diffOut, err := diffCmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	if diffOut == nil || len(diffOut) == 0 {
		if *printFlag {
			fmt.Fprintln(os.Stderr, "No changes to commit.")
		} else {
			fmt.Println("No changes to commit.")
		}
		return
	}

	logCmd := exec.Command("git", "log", "-n", "10", "--format=%s")
	logOut, err := logCmd.Output()
	if err != nil {
		logOut = []byte("")
	}

	aiMessage, err := askAi(diffOut, logOut, *ollamaFlag, *printFlag)
	if err != nil {
		log.Fatal(err)
	}

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
	return getConfigFilePath("apikey")
}

func getOllamaModelPath() (string, error) {
	return getConfigFilePath("ollama-model")
}

func getConfigFilePath(filename string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "aicommits", filename), nil
}

func saveAPIKey(key string) {
	configPath, err := getConfigPath()
	if err != nil {
		log.Fatalf("Failed to get config path: %v", err)
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}

	if err := os.WriteFile(configPath, []byte(key), 0600); err != nil {
		log.Fatalf("Failed to save API key: %v", err)
	}

	fmt.Println("API key saved successfully")
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

func askAi(diff []byte, history []byte, useOllama bool, quiet bool) (string, error) {
	prompt := buildPrompt(diff, history)
	if useOllama {
		return askOllama(prompt)
	}

	message, err := askGemini(prompt)
	if err == nil {
		return message, nil
	}

	if !quiet {
		fmt.Fprintf(os.Stderr, "Gemini unavailable, using Ollama: %v\n", err)
	}
	return askOllama(prompt)
}

func buildPrompt(diff []byte, history []byte) string {
	instruction := fmt.Sprintf(
		"Write a highly concise git commit message based on the following diff. "+
			"Output ONLY the message itself, no preamble or quotes. "+
			"STRICT LENGTH LIMIT: Keep it under 50 words total. Do NOT write long paragraphs. "+
			"Use Conventional Commits format (e.g., feat:, fix:, refactor:). "+
			"For simple changes, return ONLY ONE LINE (the header). "+
			"For complex changes, return the header and a maximum of 1-2 short bullet points. "+
			"Try to replicate the style of the last 10 commit messages:\n%s",
		history,
	)
	return fmt.Sprintf("%s\n\n%s", instruction, string(diff))
}

func askGemini(prompt string) (string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return "", err
	}

	result, err := client.Models.GenerateContent(
		ctx,
		"gemini-2.5-flash-lite",
		genai.Text(prompt),
		nil,
	)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Text()), nil
}

func askOllama(prompt string) (string, error) {
	model, err := loadOrSelectOllamaModel()
	if err != nil {
		return "", err
	}

	requestBody, err := json.Marshal(ollamaGenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return "", err
	}

	resp, err := ollamaHTTPClient.Post(ollamaURL("/api/generate"), "application/json", bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Ollama returned %s", resp.Status)
	}

	var result ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	message := strings.TrimSpace(result.Response)
	if message == "" {
		return "", fmt.Errorf("Ollama returned an empty response")
	}
	return message, nil
}

func loadOrSelectOllamaModel() (string, error) {
	modelPath, err := getOllamaModelPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(modelPath)
	if err == nil && strings.TrimSpace(string(data)) != "" {
		return strings.TrimSpace(string(data)), nil
	}

	models, err := listOllamaModels()
	if err != nil {
		return "", err
	}
	if len(models) == 0 {
		return "", fmt.Errorf("no Ollama models found. Install one with: ollama pull <model>")
	}

	fmt.Fprintln(os.Stderr, "Select Ollama model:")
	for i, model := range models {
		fmt.Fprintf(os.Stderr, "%d. %s\n", i+1, model)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(os.Stderr, "Model number: ")
		answer, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		index, err := strconv.Atoi(strings.TrimSpace(answer))
		if err == nil && index >= 1 && index <= len(models) {
			selected := models[index-1]
			if err := saveOllamaModel(modelPath, selected); err != nil {
				return "", err
			}
			return selected, nil
		}

		fmt.Fprintf(os.Stderr, "Enter a number from 1 to %d.\n", len(models))
	}
}

func listOllamaModels() ([]string, error) {
	resp, err := ollamaHTTPClient.Get(ollamaURL("/api/tags"))
	if err != nil {
		return nil, fmt.Errorf("failed to list Ollama models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Ollama returned %s while listing models", resp.Status)
	}

	var result ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse Ollama model list: %w", err)
	}

	models := make([]string, 0, len(result.Models))
	for _, model := range result.Models {
		if model.Name != "" {
			models = append(models, model.Name)
		}
	}
	sort.Strings(models)
	return models, nil
}

func saveOllamaModel(modelPath string, model string) error {
	configDir := filepath.Dir(modelPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	if err := os.WriteFile(modelPath, []byte(model), 0600); err != nil {
		return fmt.Errorf("failed to save Ollama model: %w", err)
	}
	return nil
}

func ollamaURL(path string) string {
	host := strings.TrimRight(os.Getenv("OLLAMA_HOST"), "/")
	if host == "" {
		host = "http://localhost:11434"
	}
	return host + path
}

type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
}
