package aime

import (
	"bytes"
	"crypto/tls"
	"gopkg.in/gomail.v2"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"
)

type emailSender struct {
	createReport   *template.Template
	createRevision *template.Template
	confirmIssue   *template.Template
	createIssue    *template.Template
	createAnswer   *template.Template

	dialer *gomail.Dialer
}

func NewEmailSender(host string, port int, username, password string) *emailSender {
	e := &emailSender{}
	e.dialer = gomail.NewDialer(host, port, username, password)
	e.dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	return e
}

func (e *emailSender) LoadTemplates(baseDir string) error {
	createReportBytes, err := ioutil.ReadFile(filepath.Join(baseDir, "new_report.txt"))
	if err != nil {
		return err
	}
	e.createReport, err = template.New("createReport").Parse(string(createReportBytes))
	if err != nil {
		return err
	}

	createRevisionBytes, err := ioutil.ReadFile(filepath.Join(baseDir, "new_revision.txt"))
	if err != nil {
		return err
	}
	e.createRevision, err = template.New("createRevision").Parse(string(createRevisionBytes))
	if err != nil {
		return err
	}

	confirmIssueBytes, err := ioutil.ReadFile(filepath.Join(baseDir, "confirm_issue.txt"))
	if err != nil {
		return err
	}
	e.confirmIssue, err = template.New("confirmIssue").Parse(string(confirmIssueBytes))
	if err != nil {
		return err
	}

	createIssueBytes, err := ioutil.ReadFile(filepath.Join(baseDir, "new_issue.txt"))
	if err != nil {
		return err
	}
	e.createIssue, err = template.New("createIssue").Parse(string(createIssueBytes))
	if err != nil {
		return err
	}

	createAnswerBytes, err := ioutil.ReadFile(filepath.Join(baseDir, "new_answer.txt"))
	if err != nil {
		return err
	}
	e.createAnswer, err = template.New("createAnswer").Parse(string(createAnswerBytes))
	if err != nil {
		return err
	}

	return nil
}

func (e *emailSender) createReportMail(report Report) string {
	ctx := struct {
		Report Report
	}{
		Report: report,
	}

	buf := &bytes.Buffer{}
	e.createReport.Execute(buf, ctx)

	return string(buf.Bytes())
}

func (e *emailSender) createRevisionMail(report Report, revision Revision) string {
	ctx := struct {
		Report  Report
		Version int
	}{
		Report:  report,
		Version: revision.Version,
	}

	buf := &bytes.Buffer{}
	e.createRevision.Execute(buf, ctx)

	return string(buf.Bytes())
}

func (e *emailSender) confirmIssueMail(report Report, issue Issue) string {
	ctx := struct {
		Report Report
		Issue  Issue
		Field  string
	}{
		Report: report,
		Issue:  issue,
		Field:  strings.Join(issue.Field, "."),
	}

	buf := &bytes.Buffer{}
	e.confirmIssue.Execute(buf, ctx)

	return string(buf.Bytes())
}

func (e *emailSender) createIssueMail(report Report, issue Issue) string {
	ctx := struct {
		Report Report
		Issue  Issue
		Field  string
	}{
		Report: report,
		Issue:  issue,
		Field:  strings.Join(issue.Field, "."),
	}

	buf := &bytes.Buffer{}
	e.createIssue.Execute(buf, ctx)

	return string(buf.Bytes())
}

func (e *emailSender) createAnswerMail(report Report, issue Issue, answer Answer) string {
	var token string
	if answer.Owner {
		token = issue.Token
	} else {
		token = report.Token
	}

	ctx := struct {
		Report Report
		Issue  Issue
		Answer Answer
		Field  string
		Token  string
	}{
		Report: report,
		Issue:  issue,
		Answer: answer,
		Field:  strings.Join(issue.Field, "."),
		Token:  token,
	}

	buf := &bytes.Buffer{}
	e.createAnswer.Execute(buf, ctx)

	return string(buf.Bytes())
}

func (e *emailSender) SendReportMail(report Report) error {
	txt := e.createReportMail(report)
	return e.SendMail(report.Email, "Your AIMe report", txt)
}

func (e *emailSender) SendRevisionMail(report Report, revision Revision) error {
	txt := e.createRevisionMail(report, revision)
	return e.SendMail(report.Email, "New revision of your AIMe report", txt)
}

func (e *emailSender) SendIssueConfirmationMail(report Report, issue Issue) error {
	txt := e.confirmIssueMail(report, issue)
	return e.SendMail(issue.Email, "Confirm your AIMe report issue", txt)
}

func (e *emailSender) SendIssueMail(report Report, issue Issue) error {
	txt := e.createIssueMail(report, issue)
	return e.SendMail(report.Email, "New issue in your AIMe report", txt)
}

func (e *emailSender) SendAnswerMail(report Report, issue Issue, answer Answer) error {
	txt := e.createAnswerMail(report, issue, answer)
	var to string
	if answer.Owner {
		to = issue.Email
	} else {
		to = report.Email
	}
	return e.SendMail(to, "New response in AIMe report issue", txt)
}

func (e *emailSender) SendMail(to, subject, content string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", "\"AIMe Registry\" <info@aime-registry.org>")
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)

	m.SetBody("text/plain", content)

	return e.dialer.DialAndSend(m)
}
