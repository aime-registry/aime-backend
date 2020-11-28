package aime

import (
	"encoding/json"
	"testing"
)

func TestGenerateID(t *testing.T) {
	rid := generateRandomString(1)
	if len(rid) != 1 {
		t.Fatal()
	}

	rid = generateRandomString(10)
	if len(rid) != 10 {
		t.Fatal()
	}

	if rid == generateRandomString(10) {
		t.Fatal()
	}
}

func TestIsJSON__Valid(t *testing.T) {
	validJSON := "\"hallo welt\""
	if !IsJSON(validJSON) {
		t.Fatal()
	}

	validJSON = "12"
	if !IsJSON(validJSON) {
		t.Fatal()
	}

	validJSON = "true"
	if !IsJSON(validJSON) {
		t.Fatal()
	}

	validJSON = "{\n" +
		"  \"a\": true,\n" +
		"  \"b\": [1, 2, 3]\n" +
		"}\n"
	if !IsJSON(validJSON) {
		t.Fatal()
	}
}

func TestIsJSON__Invalid(t *testing.T) {
	validJSON := "hallo"
	if IsJSON(validJSON) {
		t.Fatal()
	}

	validJSON = "12,5"
	if IsJSON(validJSON) {
		t.Fatal()
	}

	validJSON = "true false"
	if IsJSON(validJSON) {
		t.Fatal()
	}
}

func TestExtractSectionText(t *testing.T) {
	q := LoadQuestions("../../questionnaire.yaml")
	txt := ExtractSectionText(q, json.RawMessage(""), "")
	if txt != "" {
		t.Fatal()
	}

	txt = ExtractSectionText(q, json.RawMessage("{\"MD\":{\"1\":\"MyMetadata\"},\"P\":{\"1\":\"MyPurpose\"}}"), "MD")
	if txt != "MyMetadata" {
		t.Fatal()
	}

	txt = ExtractSectionText(q, json.RawMessage("{\"D\":[{\"1\":\"MyDataset1\"},{\"1\":\"MyDataset2\"}]}"), "D")
	if txt != "MyDataset1MyDataset2" {
		t.Fatal()
	}
}

func TestExtractSectionText__Keywords(t *testing.T) {
	q := LoadQuestions("../../questionnaire.yaml")
	txt := ExtractSectionText(q, json.RawMessage(""), "")
	if txt != "" {
		t.Fatal()
	}

	txt = ExtractSectionText(q, json.RawMessage("{\"MD\":{\"5\":[{\"custom\":true,\"value\":\"TEST123\"}, {\"custom\":false,\"value\":\"omics\"}]},\"P\":{\"1\":\"MyPurpose\"}}"), "MD")
	if txt != "TEST123Omics" {
		t.Fatal()
	}
}

func TestExtractField(t *testing.T) {
	q := LoadQuestions("../../questionnaire.yaml")

	txt := ExtractField(q, json.RawMessage("{\"MD\":{\"1\":\"MyMetadata\"},\"P\":{\"1\":\"MyPurpose\"}}"), []string{"MD", "1"})
	if txt != "MyMetadata" {
		t.Fatal()
	}

	txt = ExtractField(q, json.RawMessage("{\"MD\":{\"1\":\"MyMetadata\"},\"P\":{\"1\":\"MyPurpose\"}}"), []string{"P", "1"})
	if txt != "MyPurpose" {
		t.Fatal()
	}
}

func TestExtractField__List(t *testing.T) {
	q := LoadQuestions("../../questionnaire.yaml")

	txt := ExtractField(q, json.RawMessage("{\"MD\":{\"1\":\"MyMetadata\",\"6\":[{\"1\":\"Name1\",\"2\":\"bvcbvc1\"},{\"1\":\"Name2\",\"2\":\"bvcbvc2\"}]},\"P\":{\"1\":\"MyPurpose\"}}"), []string{"MD", "6", "2", "1"})
	if txt != "Name2" {
		t.Fatal(txt)
	}

	txt = ExtractField(q, json.RawMessage("{\"MD\":{\"1\":\"MyMetadata\",\"6\":[{\"1\":\"Name1\",\"2\":\"bvcbvc1\"},{\"1\":\"Name2\",\"2\":\"bvcbvc2\"}]},\"P\":{\"1\":\"MyPurpose\"}}"), []string{"MD", "6", "*", "1"})
	if txt != "Name1|Name2" {
		t.Fatal(txt)
	}

	txts := ExtractFields(q, json.RawMessage("{\"MD\":{\"1\":\"MyMetadata\",\"6\":[{\"1\":\"Name1\",\"2\":\"bvcbvc1\"},{\"1\":\"Name2\",\"2\":\"bvcbvc2\"}]},\"P\":{\"1\":\"MyPurpose\"}}"), []string{"MD", "6", "*", "1"})
	if txts[0] != "Name1" || txts[1] != "Name2" {
		t.Fatal(txt)
	}
}

func TestExtractFields(t *testing.T) {
	q := LoadQuestions("../../questionnaire.yaml")

	txt := ExtractField(q, json.RawMessage("{\"MD\":{\"5\":[{\"custom\":true,\"value\":\"a\"}]},\"P\":{\"1\":\"MyPurpose\"}}"), []string{"MD", "5"})
	if txt != "a" {
		t.Fatal()
	}

	txt = ExtractField(q, json.RawMessage("{\"MD\":{\"5\":[{\"custom\":false,\"value\":\"medical_speciality\"},{\"custom\":true,\"value\":\"b\"}]},\"P\":{\"1\":\"MyPurpose\"}}"), []string{"MD", "5"})
	if txt != "Medical speciality|b" {
		t.Fatal()
	}
}
