package aime

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestServer_StartAndShutdown(t *testing.T) {
	srv := Server{
		Port: 1234,
	}

	go srv.Start()

	time.Sleep(10 * time.Millisecond)

	srv.Shutdown()
}

func TestServer_CreateReport(t *testing.T) {
	db := &DB{Dir: "./test"}
	db.Create("")
	defer db.Delete()

	es := NewEmailSender("<EMAIL HOST>", 587, "<EMAIL USERNAME>", "<EMAIL PASSWORD>")
	es.LoadTemplates("../../templates/")

	srv := Server{
		Port: 1234,
		DB:   db,
		ES:   es,
	}

	go srv.Start()

	time.Sleep(10 * time.Millisecond)

	req := CreateReportRequest{
		Email:        "test@test.de",
		AttachReport: true,
	}
	json.Unmarshal([]byte("true"), &req.Answers)
	reqBytes, _ := json.Marshal(req)
	reqBuffer := bytes.NewBuffer(reqBytes)

	resp, _ := http.Post("http://127.0.0.1:1234/report", "application/json", reqBuffer)
	if resp.StatusCode != 200 {
		t.Fatal()
	}

	respBytes, _ := ioutil.ReadAll(resp.Body)
	respStruct := CreateRevisionResponse{}
	json.Unmarshal(respBytes, &respStruct)
	if respStruct.ID == "" {
		t.Fatal()
	}
	if respStruct.Version != 1 {
		t.Fatal()
	}
}

func TestServer_CreateRevision(t *testing.T) {
	db := &DB{Dir: "./test"}
	db.Create("")
	defer db.Delete()

	es := NewEmailSender("<EMAIL HOST>", 587, "<EMAIL USERNAME>", "<EMAIL PASSWORD>")
	es.LoadTemplates("../../templates/")

	srv := Server{
		Port: 1234,
		DB:   db,
		ES:   es,
	}

	go srv.Start()

	rp := srv.DB.CreateReport("", true)
	srv.DB.CreateRevision(rp.ID, "", json.RawMessage("true"), rp.Token, true)

	time.Sleep(10 * time.Millisecond)

	req := CreateRevisionRequest{
		Email:        "test@test.de",
		AttachReport: true,
		Password:     rp.Token,
	}
	json.Unmarshal([]byte("true"), &req.Answers)
	reqBytes, _ := json.Marshal(req)
	reqBuffer := bytes.NewBuffer(reqBytes)

	c := http.Client{}

	r, _ := http.NewRequest("PUT", "http://127.0.0.1:1234/report/"+rp.ID, reqBuffer)
	resp, _ := c.Do(r)
	if resp.StatusCode != 200 {
		t.Fatal()
	}

	respBytes, _ := ioutil.ReadAll(resp.Body)
	respStruct := CreateRevisionResponse{}
	json.Unmarshal(respBytes, &respStruct)
	if respStruct.ID != rp.ID {
		t.Fatal()
	}
	if respStruct.Version != 2 {
		t.Fatal()
	}
}

func TestServer_CreateIssue(t *testing.T) {
	db := &DB{Dir: "./test"}
	db.Create("")
	defer db.Delete()

	es := NewEmailSender("<EMAIL HOST>", 587, "<EMAIL USERNAME>", "<EMAIL PASSWORD>")
	es.LoadTemplates("../../templates/")

	srv := Server{
		Port: 1234,
		DB:   db,
		ES:   es,
	}

	go srv.Start()

	rp := srv.DB.CreateReport("", true)
	srv.DB.CreateRevision(rp.ID, "", json.RawMessage("true"), rp.Token, true)

	time.Sleep(10 * time.Millisecond)

	req := CreateIssueRequest{
		Name:    "J. Matschinske",
		Email:   "test@test.de",
		Field:   nil,
		Content: "Hallo Welt!",
		Type:    1,
	}
	reqBytes, _ := json.Marshal(req)
	reqBuffer := bytes.NewBuffer(reqBytes)

	c := http.Client{}

	r, _ := http.NewRequest("POST", "http://127.0.0.1:1234/report/"+rp.ID+"/comment", reqBuffer)
	resp, _ := c.Do(r)
	if resp.StatusCode != 200 {
		t.Fatal()
	}

	respBytes, _ := ioutil.ReadAll(resp.Body)
	respStruct := CreateIssueResponse{}
	json.Unmarshal(respBytes, &respStruct)
	if respStruct.ID != 1 {
		t.Fatal()
	}
	if respStruct.Password == "" {
		t.Fatal()
	}

	rp = srv.DB.GetReport(rp.ID)
	if rp.Comments != 1 {
		t.Fatal()
	}

	r, _ = http.NewRequest("GET", "http://127.0.0.1:1234/report/"+rp.ID, reqBuffer)
	resp, _ = c.Do(r)
	if resp.StatusCode != 200 {
		t.Fatal()
	}

	respBytes, _ = ioutil.ReadAll(resp.Body)
	respStruct2 := GetRevisionResponse{}
	json.Unmarshal(respBytes, &respStruct2)
	if len(respStruct2.Issues) != 0 {
		t.Fatal()
	}
}

func TestServer_CreateAnswer(t *testing.T) {
	db := &DB{Dir: "./test"}
	db.Create("")
	defer db.Delete()

	es := NewEmailSender("<EMAIL HOST>", 587, "<EMAIL USERNAME>", "<EMAIL PASSWORD>")
	es.LoadTemplates("../../templates/")

	srv := Server{
		Port: 1234,
		DB:   db,
		ES:   es,
	}

	go srv.Start()

	rp := srv.DB.CreateReport("", true)

	time.Sleep(10 * time.Millisecond)

	com := srv.DB.CreateIssue(rp.ID, "Name", "email", nil, "Hallo Welt", 1)

	time.Sleep(10 * time.Millisecond)

	req := CreateAnswerRequest{
		Content: "Hallo Welt!",
	}
	reqBytes, _ := json.Marshal(req)
	reqBuffer := bytes.NewBuffer(reqBytes)

	c := http.Client{}

	r, _ := http.NewRequest("POST", "http://127.0.0.1:1234/report/"+rp.ID+"/comment/"+strconv.Itoa(com.ID), reqBuffer)
	resp, _ := c.Do(r)
	if resp.StatusCode != 200 {
		t.Fatal()
	}

	respBytes, _ := ioutil.ReadAll(resp.Body)
	respStruct := CreateAnswerResponse{}
	json.Unmarshal(respBytes, &respStruct)
	if respStruct.ID != 1 {
		t.Fatal()
	}

	com = srv.DB.GetIssue(rp.ID, com.ID)
	if len(com.Answers) != 1 {
		t.Fatal()
	}
}
