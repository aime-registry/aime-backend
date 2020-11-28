package aime

import (
	"strings"
	"testing"
	"time"
)

func TestEmailSender_LoadTemplates(t *testing.T) {
	es := emailSender{}
	err := es.LoadTemplates("../../templates/")
	if err != nil {
		t.Fatal(err)
	}
}

func TestEmailSender_createReportMail(t *testing.T) {
	es := emailSender{}
	es.LoadTemplates("../../templates/")

	text := es.createReportMail(Report{
		ID:    "MyTestID",
		Token: "MyTestToken",
	})

	// Ensure it contains the report URL
	if !strings.Contains(text, "https://aime.report/MyTestID") {
		t.Fatal()
	}

	// Ensure it contains the admin link
	if !strings.Contains(text, "https://aime-registry.org/questionnaire?id=MyTestID&p=MyTestToken") {
		t.Fatal()
	}
}

func TestEmailSender_SendReportMail(t *testing.T) {
	es := NewEmailSender("<EMAIL HOST>", 587, "<EMAIL USERNAME>", "<EMAIL PASSWORD>")
	es.LoadTemplates("../../templates/")

	es.SendReportMail(Report{
		Email: "test@test.de",
		ID:    "MyTestID",
		Token: "MyTestToken",
	})
}

func TestEmailSender_confirmIssueMail(t *testing.T) {
	es := emailSender{}
	es.LoadTemplates("../../templates/")

	text := es.confirmIssueMail(Report{
		ID:    "MyTestID",
		Token: "MyTestToken",
	}, Issue{
		ID:    1,
		Name:  "Mein Name",
		Field: []string{"A", "B", "C"},
		Token: "geheim01",
	})

	// Ensure it contains the field
	if !strings.Contains(text, "A.B.C") {
		t.Fatal()
	}

	// Ensure it contains the admin link
	if !strings.Contains(text, "https://aime-registry.org/report/MyTestID/issue/1?p=geheim01&confirm=1") {
		t.Fatal()
	}
}

func TestEmailSender_createAnswerMail__Owner(t *testing.T) {
	es := emailSender{}
	es.LoadTemplates("../../templates/")

	text := es.createAnswerMail(Report{
		ID:    "MyTestID",
		Token: "MyTestToken",
	}, Issue{
		ID:    1,
		Name:  "Mein Name",
		Field: []string{"A", "B", "C"},
		Token: "geheim01",
	}, Answer{
		ID:        0,
		CreatedAt: time.Time{},
		Content:   "Test123123",
		Owner:     true,
	})

	if strings.Contains(text, "MyTestToken") {
		t.Fatal()
	}
	if !strings.Contains(text, "geheim01") {
		t.Fatal()
	}
}

func TestEmailSender_createAnswerMail__Questioner(t *testing.T) {
	es := emailSender{}
	es.LoadTemplates("../../templates/")

	text := es.createAnswerMail(Report{
		ID:    "MyTestID",
		Token: "MyTestToken",
	}, Issue{
		ID:    1,
		Name:  "Mein Name",
		Field: []string{"A", "B", "C"},
		Token: "geheim01",
	}, Answer{
		ID:        0,
		CreatedAt: time.Time{},
		Content:   "Test123123",
		Owner:     false,
	})

	if strings.Contains(text, "geheim01") {
		t.Fatal()
	}
	if !strings.Contains(text, "MyTestToken") {
		t.Fatal()
	}
}
