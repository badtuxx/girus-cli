package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProgress_AddLab(t *testing.T) {
	p := &Progress{}
	p.AddLab("lab1", "concluded")

	if len(p.Labs) != 1 {
		t.Errorf("Esperado 1 lab, temos %d", len(p.Labs))
	}

	if p.Labs[0].Name != "lab1" {
		t.Errorf("Esperado lab com nome 'lab1', temos '%s'", p.Labs[0].Name)
	}

	if p.Labs[0].Status != "concluded" {
		t.Errorf("Esperado lab com status 'concluded', temos '%s'", p.Labs[0].Status)
	}
}

func TestProgress_GetLab(t *testing.T) {
	p := &Progress{
		Labs: []Lab{
			{Name: "lab1", Status: "concluded"},
			{Name: "lab2", Status: "in-progress"},
		},
	}

	lab := p.GetLab("lab1")
	if lab == nil {
		t.Error("Esperado encontrar lab1")
	} else if lab.Name != "lab1" {
		t.Errorf("Esperado lab com nome 'lab1', temos '%s'", lab.Name)
	}

	lab = p.GetLab("lab3")
	if lab != nil {
		t.Error("Esperado não encontrar o lab3")
	}
}

func TestProgress_SetLabStatus(t *testing.T) {
	p := &Progress{
		Labs: []Lab{
			{Name: "lab1", Status: "in-progress"},
		},
	}

	err := p.SetLabStatus("lab1", "concluded")
	if err != nil {
		t.Errorf("Erro não esperado: %v", err)
	}

	if p.Labs[0].Status != "concluded" {
		t.Errorf("Esperado status 'concluded', temos '%s'", p.Labs[0].Status)
	}

	err = p.SetLabStatus("lab2", "concluded")
	if err == nil {
		t.Error("Esperado encontrar um erro ao setar o status de um lab que não existe")
	}
}

func TestProgress_SaveAndLoad(t *testing.T) {
	// Setup temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "girus-test")
	if err != nil {
		t.Fatalf("Falha ao criar o diretório temporário: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "progresso.yaml")

	// Create a new Progress instance with the temp file path
	p := NewProgress(tmpFile)
	p.AddLab("lab1", "concluded")

	// Save to file
	err = p.SaveProgressToFile()
	if err != nil {
		t.Fatalf("Falha ao salvar o progresso: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Não foi possível criar o arquivo")
	}

	// Load from file into a new instance
	p2 := NewProgress(tmpFile)
	err = p2.LoadProgressFromFile()
	if err != nil {
		t.Fatalf("Falha ao carregar o progresso: %v", err)
	}

	if len(p2.Labs) != 1 {
		t.Errorf("Esperado 1 lab, temos %d", len(p2.Labs))
	}

	if p2.Labs[0].Name != "lab1" {
		t.Errorf("Esperado lab com nome 'lab1', temos '%s'", p2.Labs[0].Name)
	}
}
