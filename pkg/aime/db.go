package aime

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type keyword struct {
	keyword string
	reports []string
}

type category struct {
	category string
	reports  []string
}

type DB struct {
	Dir           string
	KeywordGroups KeywordGroups

	questions Question

	keywordSet  map[string]*keyword
	categorySet map[string]*category
	mutex       sync.Mutex
}

type Report struct {
	ID        string    `json:"id"`
	Email     string    `json:"-"`
	Token     string    `json:"-"`
	Revisions int       `json:"revisions"`
	Comments  int       `json:"comments"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Public    bool      `json:"public"`
}

type UnsafeReport struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	Revisions int       `json:"revisions"`
	Comments  int       `json:"comments"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Public    bool      `json:"public"`
}

// Ensure they contain the same fields
var _ = Report(UnsafeReport{})
var _ = UnsafeReport(Report{})

type Revision struct {
	ReportID  string          `json:"reportId"`
	Version   int             `json:"version"`
	Answers   json.RawMessage `json:"answers"`
	CreatedAt time.Time       `json:"createdAt"`
	Public    bool            `json:"public"`
}

type Answer struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Content   string    `json:"content"`
	Owner     bool      `json:"owner"`
}

type Issue struct {
	ID         int    `json:"id"`
	ReportID   string `json:"reportId"`
	RevisionID int    `json:"revisionId"`

	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"createdAt"`
	VerifiedAt time.Time `json:"-"`
	Type       int       `json:"type"`
	Field      []string  `json:"field"`
	Content    string    `json:"content"`

	Answers []Answer `json:"answers"`

	Verified bool `json:"confirmed"`
	Deleted  bool `json:"-"`

	Email string `json:"-"`
	Token string `json:"-"`
}

type UnsafeIssue struct {
	ID         int    `json:"id"`
	ReportID   string `json:"reportId"`
	RevisionID int    `json:"revisionId"`

	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"createdAt"`
	VerifiedAt time.Time `json:"verifiedAt"`
	Type       int       `json:"type"`
	Field      []string  `json:"field"`
	Content    string    `json:"content"`

	Answers []Answer `json:"answers"`

	Verified bool `json:"verified"`
	Deleted  bool `json:"deleted"`

	Email string `json:"email"`
	Token string `json:"token"`
}

// Ensure they contain the same fields
var _ = Issue(UnsafeIssue{})
var _ = UnsafeIssue(Issue{})

type KeywordGroup struct {
	Sources       []string `yaml:"sources"`
	Targets       []string `yaml:"targets"`
	CaseSensitive bool     `yaml:"caseSensitive"`
}

type KeywordGroups []KeywordGroup

const pendingTime = 2 * 7 * 24 * time.Hour

func (c Issue) Pending() bool {
	if c.VerifiedAt.IsZero() {
		return true
	}
	for _, a := range c.Answers {
		if a.Owner {
			return false
		}
	}
	return time.Now().Sub(c.VerifiedAt) < pendingTime
}

// DB

func (db *DB) Create(quFilename string) {
	os.MkdirAll(filepath.Join(db.Dir, "reports"), os.ModePerm)
	os.MkdirAll(filepath.Join(db.Dir, "documents"), os.ModePerm)

	if quFilename != "" {
		db.questions = LoadQuestions(quFilename)
	}
}

func (db *DB) Delete() {
	os.RemoveAll(db.Dir)
}

// Report

func (db *DB) CreateReport(email string, public bool) *Report {
	id := generateRandomString(6)

	rpPath := db.reportPath(id)
	os.MkdirAll(rpPath, os.ModePerm)

	rp := &Report{
		ID:        id,
		Email:     email,
		Token:     generateRandomString(16),
		Revisions: 0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Public:    public,
	}

	db.SetReport(*rp)

	return rp
}

func (db *DB) ExistsReport(id string) bool {
	rpPath := db.reportPath(id)
	s, err := os.Stat(rpPath)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func (db *DB) GetReport(id string) *Report {
	rpPath := db.reportPath(id)
	rpBytes, err := ioutil.ReadFile(filepath.Join(rpPath, "report.json"))
	if err != nil {
		return nil
	}
	rp := UnsafeReport{}
	err = json.Unmarshal(rpBytes, &rp)
	if err != nil {
		return nil
	}
	srp := Report(rp)
	return &srp
}

func (db *DB) SetReport(rp Report) {
	if rp.Token == "" {
		panic("no token")
	}

	rpBytes, _ := json.Marshal(UnsafeReport(rp))
	ioutil.WriteFile(filepath.Join(db.reportPath(rp.ID), "report.json"), rpBytes, os.ModePerm)
}

func (db *DB) DeleteReport(id string) {
	os.RemoveAll(db.reportPath(id))
}

// Search

func (db *DB) GetLatestRevisions(includeHidden bool) chan *Revision {
	r := make(chan *Revision)
	go func(r chan *Revision, ih bool) {
		defer close(r)
		files, err := ioutil.ReadDir(filepath.Join(db.Dir, "reports"))
		if err != nil {
			return
		}
		for _, f := range files {
			if !f.IsDir() {
				continue
			}
			id := f.Name()
			rev := db.LatestRevision(id)
			if rev == nil {
				continue
			}
			if ih || rev.Public {
				r <- rev
			}
		}
	}(r, includeHidden)
	return r
}

// Keywords

func cleanWord(a string) string {
	if len(a) <= 1 {
		return a
	}

	if strings.ToUpper(a) == a {
		return a
	}

	if strings.ToLower(a) == a {
		return a
	}

	if strings.ToLower(a[1:]) == a[1:] {
		return strings.ToLower(a)
	}

	return a
}

func clean(w string) string {
	ws := strings.Split(w, " ")
	var ws2 []string
	for _, w := range ws {
		ws2 = append(ws2, cleanWord(w))
	}
	return strings.Join(ws2, " ")
}

func (kwg KeywordGroups) transform(sources []string) []string {
	if kwg == nil {
		return sources
	}
	targetMap := map[string]bool{}
	for _, s1 := range sources {
		replaced := false
		for _, g := range kwg {
			s1i := s1
			if !g.CaseSensitive {
				s1i = strings.ToLower(s1i)
			}
			for _, s2 := range g.Sources {
				if !g.CaseSensitive {
					s2 = strings.ToLower(s2)
				}
				if s1i == s2 {
					for _, t := range g.Targets {
						targetMap[t] = true
					}
					replaced = true
				}
			}
		}
		if !replaced {
			targetMap[clean(s1)] = true
		}
	}
	targets := []string{}
	for t := range targetMap {
		targets = append(targets, t)
	}
	return targets
}

func (db *DB) BuildKeywordList(fk []string, fc []string) (int, int) {
	db.mutex.Lock()

	c := db.GetLatestRevisions(false)

	db.keywordSet = map[string]*keyword{}
	db.categorySet = map[string]*category{}

	for rev := range c {
		ks := db.KeywordGroups.transform(ExtractFields(db.questions, rev.Answers, fk))
		for _, kw := range ks {
			if kp, ok := db.keywordSet[kw]; ok {
				kp.reports = append(kp.reports, rev.ReportID)
				db.keywordSet[kw] = kp
			} else {
				db.keywordSet[kw] = &keyword{
					keyword: kw,
					reports: []string{rev.ReportID},
				}
			}
		}
		c := ExtractField(db.questions, rev.Answers, fc)
		if c != "" {
			if kp, ok := db.categorySet[c]; ok {
				kp.reports = append(kp.reports, rev.ReportID)
				db.categorySet[c] = kp
			} else {
				db.categorySet[c] = &category{
					category: c,
					reports:  []string{rev.ReportID},
				}
			}
		}
	}

	db.mutex.Unlock()

	return len(db.keywordSet), len(db.categorySet)
}

func (db *DB) GetKeyword(k string) *keyword {
	return db.keywordSet[k]
}

func (db *DB) GetCategory(k string) *category {
	return db.categorySet[k]
}

// Revision

func (db *DB) CreateRevision(id string, email string, answers json.RawMessage, password string, public bool) *Revision {
	rp := db.GetReport(id)
	if rp == nil {
		return nil
	}

	if rp.Token != password {
		return nil
	}

	os.MkdirAll(db.revisionPath(id), os.ModePerm)

	ver := rp.Revisions + 1

	revPath := db.revisionFilePath(id, ver)

	rev := &Revision{
		ReportID:  id,
		Version:   ver,
		Answers:   answers,
		CreatedAt: time.Now(),
		Public:    public,
	}

	revBytes, _ := json.Marshal(rev)

	ioutil.WriteFile(revPath, revBytes, os.ModePerm)

	rp.Email = email
	rp.Revisions = ver
	rp.Public = public

	db.SetReport(*rp)

	go db.BuildKeywordList([]string{"MD", "5"}, []string{"P", "3", "1"})

	return rev
}

func ReadKeywordGroups(filename string) KeywordGroups {
	var kwg KeywordGroups

	kwgBytes, _ := ioutil.ReadFile(filename)

	yaml.Unmarshal(kwgBytes, &kwg)

	return kwg
}

func (db *DB) GetKeywords() []*keyword {
	var k []*keyword
	for _, kw := range db.keywordSet {
		k = append(k, kw)
	}
	sort.Slice(k, func(i, j int) bool {
		return len(k[j].reports) < len(k[i].reports)
	})
	return k
}

func (db *DB) ExistsRevision(id string, ver int) bool {
	return false
}

func (db *DB) GetRevision(id string, ver int) *Revision {
	revPath := db.revisionFilePath(id, ver)
	revBytes, err := ioutil.ReadFile(revPath)
	if err != nil {
		return nil
	}
	rev := &Revision{}
	err = json.Unmarshal(revBytes, &rev)
	if err != nil {
		return nil
	}
	return rev
}

func (db *DB) LatestRevision(id string) *Revision {
	rp := db.GetReport(id)
	if rp == nil {
		return nil
	}
	return db.GetRevision(id, rp.Revisions)
}

// Document

func (db *DB) ReadDocument(fileName string) []byte {
	fileBytes, err := ioutil.ReadFile(db.documentPath(fileName))
	if err != nil {
		return nil
	}
	return fileBytes
}

func (db *DB) UploadDocument(fileBytes []byte) string {
	fileName := generateRandomString(12) + ".pdf"
	_ = ioutil.WriteFile(db.documentPath(fileName), fileBytes, os.ModePerm)
	return fileName
}

// Issue

func (db *DB) CreateIssue(id string, name string, email string, field []string, content string, cType int) *Issue {
	rep := db.GetReport(id)
	if rep == nil {
		return nil
	}

	os.MkdirAll(db.commentPath(id), os.ModePerm)

	cid := rep.Comments + 1

	c := Issue{
		ID:         cid,
		ReportID:   id,
		RevisionID: rep.Revisions,
		Name:       name,
		CreatedAt:  time.Now(),
		Type:       cType,
		Field:      field,
		Content:    content,
		Verified:   false,
		Deleted:    false,
		Email:      email,
		Token:      generateRandomString(16),
	}

	db.SetIssue(c)

	rep.Comments = cid

	db.SetReport(*rep)

	return &c
}

func (db *DB) ValidateIssue(id string, comment int, token string) bool {
	c := db.GetIssue(id, comment)
	if c == nil {
		return false
	}
	if c.Token != token {
		return false
	}

	c.Verified = true
	c.VerifiedAt = time.Now()

	db.SetIssue(*c)

	return true
}

func (db *DB) CreateAnswer(id string, comment int, content string, token string) *Answer {
	rep := db.GetReport(id)
	if rep == nil {
		return nil
	}

	c := db.GetIssue(id, comment)
	if c == nil {
		return nil
	}

	var owner bool

	if token == c.Token {
		owner = false
	} else if token == rep.Token {
		owner = true
	} else {
		return nil
	}

	a := Answer{
		ID:        len(c.Answers) + 1,
		CreatedAt: time.Now(),
		Content:   content,
		Owner:     owner,
	}

	c.Answers = append(c.Answers, a)

	db.SetIssue(*c)

	return &a
}

func (db *DB) ValidateAnswer(id string, comment int, token string) {
	panic("implement me")
}

func (db *DB) GetIssue(id string, comment int) *Issue {
	comPath := db.commentFilePath(id, comment)
	comBytes, err := ioutil.ReadFile(comPath)
	if err != nil {
		return nil
	}
	c := UnsafeIssue{}
	err = json.Unmarshal(comBytes, &c)
	if err != nil {
		return nil
	}
	sc := Issue(c)
	return &sc
}

func (db *DB) SetIssue(c Issue) {
	comBytes, _ := json.Marshal(UnsafeIssue(c))
	ioutil.WriteFile(db.commentFilePath(c.ReportID, c.ID), comBytes, os.ModePerm)
}

func (db *DB) GetReportIssues(reportID string, includePending bool) chan *Issue {
	ic := make(chan *Issue)
	go func(ic chan *Issue, rid string, ip bool) {
		defer close(ic)
		files, err := ioutil.ReadDir(db.commentPath(rid))
		if err != nil {
			return
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			id, err := strconv.Atoi(strings.Split(f.Name(), ".")[0])
			if err != nil {
				continue
			}
			iss := db.GetIssue(rid, id)
			if iss == nil {
				continue
			}
			if iss.Deleted || !iss.Verified {
				continue
			}
			if ip || !iss.Pending() {
				ic <- iss
			}
		}
	}(ic, reportID, includePending)
	return ic
}

// Join consortium

func (db *DB) AddContribution(answers []byte) {
	os.MkdirAll(db.contributionPath(), os.ModePerm)
	ioutil.WriteFile(db.contributionFilePath(), answers, os.ModePerm)
}

// Private functions

func (db *DB) documentPath(name string) string {
	return filepath.Join(db.Dir, "documents", name)
}

func (db *DB) reportPath(id string) string {
	return filepath.Join(db.Dir, "reports", id)
}

func (db *DB) revisionPath(id string) string {
	return filepath.Join(db.Dir, "reports", id, "revisions")
}

func (db *DB) revisionFilePath(id string, ver int) string {
	return filepath.Join(db.revisionPath(id), fmt.Sprintf("%04d.json", ver))
}

func (db *DB) commentPath(id string) string {
	return filepath.Join(db.Dir, "reports", id, "comments")
}

func (db *DB) commentFilePath(id string, com int) string {
	return filepath.Join(db.commentPath(id), fmt.Sprintf("%04d.json", com))
}

func (db *DB) contributionPath() string {
	return filepath.Join(db.Dir, "contributions")
}

func (db *DB) contributionFilePath() string {
	return filepath.Join(db.contributionPath(), fmt.Sprintf("%d.json", time.Now().UnixNano()))
}
