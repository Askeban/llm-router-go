package main

import (
	"fmt"
	"github.com/Askeban/llm-router-go/internal/models"
)

func main() {
	model := models.ModelProfile{
		ID: "test",
		Provider: "test",
		DisplayName: "Test Model",
	}
	
	fmt.Printf("Model: %s\n", model.ID)
}