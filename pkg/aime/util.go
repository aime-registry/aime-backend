package aime

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strconv"
	"strings"
)

type Config struct {
	Options []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"options"`
}

type Question struct {
	ID       string     `json:"id"`
	Type     string     `json:"type"`
	Title    string     `json:"title"`
	Question string     `json:"question"`
	Config   *Config    `json:"config"`
	Children []Question `json:"children"`
	Child    *Question  `json:"child"`
}

func IsJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

func generateRandomString(n int) string {
	letterRunes := []rune("1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		rb := [4]byte{}
		rand.Read(rb[:])
		ri := int(binary.LittleEndian.Uint32(rb[:]))
		if ri < 0 {
			ri = -ri
		}
		letter := ri % len(letterRunes)
		b[i] = letterRunes[letter]
	}

	return string(b)
}

func extractValue(a interface{}, q Question) string {
	val, ok := a.(map[string]interface{})
	if !ok {
		return ""
	}
	c, ok := val["custom"]
	if !ok {
		return ""
	}
	cb, ok := c.(bool)
	if !ok {
		return ""
	}
	value, ok := val["value"]
	if !ok {
		return ""
	}
	valStr, ok := value.(string)
	if !ok {
		return ""
	}
	if cb {
		return valStr
	}
	if q.Config == nil {
		return ""
	}
	for _, e := range q.Config.Options {
		if e.Key == valStr {
			return e.Value
		}
	}
	return ""
}

func extractField(q Question, a interface{}, ids []string, sep string) string {
	if a == nil {
		return ""
	}

	if q.Type == "string" || q.Type == "text" {
		if len(ids) != 0 {
			return ""
		}
		return a.(string)
	}

	if q.Type == "select" || q.Type == "radio" {
		if len(ids) != 0 {
			return ""
		}
		return extractValue(a, q)
	}

	if q.Type == "checkboxes" || q.Type == "tags" {
		if len(ids) != 0 {
			return ""
		}
		vals, ok := a.([]interface{})
		if !ok {
			return ""
		}
		txt := ""
		for _, v := range vals {
			val := extractValue(v, q)
			if txt != "" {
				txt += sep
			}
			txt += val
		}
		return txt
	}

	if q.Type == "complex" {
		if len(ids) == 0 {
			return ""
		}
		if q.Children == nil {
			return ""
		}
		compl, ok := a.(map[string]interface{})
		if !ok {
			return ""
		}
		for _, child := range q.Children {
			if child.ID == ids[0] {
				return extractField(child, compl[ids[0]], ids[1:], sep)
			}
		}
		return ""
	}

	if q.Type == "list" {
		if len(ids) == 0 {
			return ""
		}
		if q.Child == nil {
			return ""
		}
		list, ok := a.([]interface{})
		if !ok {
			return ""
		}
		if ids[0] == "*" {
			txt := ""
			for _, a := range list {
				if txt != "" {
					txt += sep
				}
				txt += extractField(*q.Child, a, ids[1:], sep)
			}
			return txt
		} else {
			fld, err := strconv.Atoi(ids[0])
			if err != nil || fld <= 1 {
				return ""
			}
			return extractField(*q.Child, list[fld-1], ids[1:], sep)
		}
	}

	return ""
}

func extractText(q Question, a interface{}) string {
	if a == nil {
		return ""
	}

	if q.Type == "string" || q.Type == "text" {
		return a.(string)
	}

	if q.Type == "select" || q.Type == "radio" {
		return extractValue(a, q)
	}

	if q.Type == "checkboxes" || q.Type == "tags" {
		vals, ok := a.([]interface{})
		if !ok {
			return ""
		}
		txt := ""
		for _, v := range vals {
			val := extractValue(v, q)
			txt += val
		}
		return txt
	}

	if q.Type == "complex" {
		if q.Children == nil {
			return ""
		}
		compl, ok := a.(map[string]interface{})
		if !ok {
			return ""
		}
		txt := ""
		for _, child := range q.Children {
			txt += extractText(child, compl[child.ID])
		}
		return txt
	}

	if q.Type == "list" {
		if q.Child == nil {
			return ""
		}
		list, ok := a.([]interface{})
		if !ok {
			return ""
		}
		txt := ""
		for _, ae := range list {
			txt += extractText(*q.Child, ae)
		}
		return txt
	}

	return ""
}

func LoadQuestions(filename string) Question {
	q := Question{}
	qBytes, _ := ioutil.ReadFile(filename)
	yaml.Unmarshal(qBytes, &q)
	return q
}

func ExtractField(q Question, answers json.RawMessage, ids []string) string {
	var ans interface{}
	err := json.Unmarshal(answers, &ans)
	if err != nil {
		return ""
	}

	return extractField(q, ans, ids, "|")
}

func ExtractFields(q Question, answers json.RawMessage, ids []string) []string {
	var ans interface{}
	err := json.Unmarshal(answers, &ans)
	if err != nil {
		return nil
	}

	sep := "|.#.|"

	return strings.Split(extractField(q, ans, ids, sep), sep)
}

func ExtractSectionText(question Question, answers json.RawMessage, section string) string {
	var ans interface{}
	err := json.Unmarshal(answers, &ans)
	if err != nil {
		return ""
	}

	sections, ok := ans.(map[string]interface{})
	if !ok {
		return ""
	}

	for _, ch := range question.Children {
		if ch.ID == section {
			return extractText(ch, sections[section])
		}
	}

	return ""
}
