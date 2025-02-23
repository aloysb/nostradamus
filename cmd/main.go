package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"nostradamus/internal/logger"
	"nostradamus/internal/llm"
)

func main() {
	if len(os.Args) < 2 {
		logger.Error("No input provided", "usage", "go run main.go <input>")
		os.Exit(1)
	}
	input := strings.Join(os.Args[1:], " ")
	logger.Info("Received input", "input", input)

	llmClient, err := llm.NewClient(http.DefaultClient)
	if err != nil {
		logger.Error("Error creating LLM client", "error", err)
		os.Exit(1)
	}
	result, err := llm.GenerateCritiquedPredictions(input, llmClient)
	if err != nil {
		logger.Error("Error generating critiqued predictions", "error", err)
		os.Exit(1)
	}
	logger.Info("Final valid critiqued predictions", "result", result)
	fmt.Println(result)
}
