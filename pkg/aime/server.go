package aime

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	Port int
	DB   *DB
	ES   *emailSender

	srv *http.Server
}

func (s *Server) serveRevision(w http.ResponseWriter, id string, ver int) {
	rp := s.DB.GetReport(id)
	if rp == nil {
		w.WriteHeader(404)
		return
	}
	rev := s.DB.GetRevision(id, ver)
	if rev == nil {
		w.WriteHeader(404)
		return
	}

	resp := GetRevisionResponse{
		Answers:  rev.Answers,
		Revision: rev.Version,
		Public:   rev.Public,
	}

	for i := 1; i <= rp.Revisions; i++ {
		oldRev := s.DB.GetRevision(id, i)
		resp.Revisions = append(resp.Revisions, RevisionInfo{
			Revision:  oldRev.Version,
			CreatedAt: oldRev.CreatedAt,
		})
	}

	for c := range s.DB.GetReportIssues(rp.ID, false) {
		resp.Issues = append(resp.Issues, IssueInfo{
			ID:         c.ID,
			RevisionID: c.RevisionID,
			CreatedAt:  c.CreatedAt,
			Name:       c.Name,
			Type:       c.Type,
		})
	}

	respBytes, _ := json.Marshal(resp)

	w.Write(respBytes)
}

func (s *Server) Start() {
	r := mux.NewRouter()

	r.HandleFunc("/report/{id}/issue", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		id := vars["id"]
		if id == "" {
			w.WriteHeader(404)
			return
		}

		if r.Method == "POST" {
			reqBytes, _ := ioutil.ReadAll(r.Body)

			req := CreateIssueRequest{}

			json.Unmarshal(reqBytes, &req)

			if req.Email == "" || req.Name == "" || req.Content == "" {
				w.WriteHeader(400)
				return
			}

			rp := s.DB.GetReport(id)
			if rp == nil {
				w.WriteHeader(400)
				return
			}

			com := s.DB.CreateIssue(rp.ID, req.Name, req.Email, req.Field, req.Content, req.Type)
			if com == nil {
				w.WriteHeader(401)
				return
			}

			go func(rp Report, com Issue) {
				s.ES.SendIssueConfirmationMail(rp, com)
			}(*rp, *com)

			resp := CreateIssueResponse{
				ID:       com.ID,
				Password: com.Token,
			}

			respBytes, _ := json.Marshal(resp)

			_, _ = w.Write(respBytes)

			return
		}
	}).Methods("POST")

	r.HandleFunc("/report/{id}/issue/{issue}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		id := vars["id"]
		if id == "" {
			w.WriteHeader(404)
			return
		}

		issStr := vars["issue"]
		if issStr == "" {
			w.WriteHeader(404)
			return
		}

		iid, err := strconv.Atoi(issStr)
		if err != nil {
			w.WriteHeader(404)
			return
		}

		rp := s.DB.GetReport(id)
		if rp == nil {
			w.WriteHeader(400)
			return
		}

		iss := s.DB.GetIssue(id, iid)
		if iss == nil || iss.Deleted {
			w.WriteHeader(404)
			return
		}

		if r.Method == "GET" {
			pw := r.URL.Query().Get("p")

			if pw == "" {
				if !iss.Verified || iss.Pending() {
					w.WriteHeader(403)
					return
				}
			} else {
				if !iss.Verified && pw != iss.Token {
					w.WriteHeader(403)
					return
				}
				if iss.Pending() && pw != iss.Token && pw != rp.Token {
					w.WriteHeader(403)
					return
				}
			}

			respStruct := struct {
				ID         int    `json:"id"`
				ReportID   string `json:"reportId"`
				RevisionID int    `json:"revisionId"`

				Name      string    `json:"name"`
				CreatedAt time.Time `json:"createdAt"`
				Type      int       `json:"type"`
				Field     []string  `json:"field"`
				Content   string    `json:"content"`

				Answers []Answer `json:"answers"`

				Verified bool `json:"confirmed"`
				Pending  bool `json:"pending"`

				PendingUntil time.Time `json:"pendingUntil"`
			}{
				iss.ID,
				iss.ReportID,
				iss.RevisionID,
				iss.Name,
				iss.CreatedAt,
				iss.Type,
				iss.Field,
				iss.Content,
				iss.Answers,
				iss.Verified,
				iss.Pending(),
				iss.VerifiedAt.Add(pendingTime),
			}

			issBytes, _ := json.Marshal(respStruct)

			w.Write(issBytes)
		} else if r.Method == "POST" {
			reqBytes, _ := ioutil.ReadAll(r.Body)

			req := CreateAnswerRequest{}

			json.Unmarshal(reqBytes, &req)

			if req.Content == "" {
				w.WriteHeader(400)
				return
			}

			if iss.Deleted || !iss.Verified {
				w.WriteHeader(404)
				return
			}

			a := s.DB.CreateAnswer(id, iid, req.Content, r.URL.Query().Get("p"))
			if a == nil {
				w.WriteHeader(401)
				return
			}

			go func(rp Report, com Issue, a Answer) {
				s.ES.SendAnswerMail(rp, com, a)
			}(*rp, *iss, *a)

			resp := CreateAnswerResponse{
				ID: a.ID,
			}

			respBytes, _ := json.Marshal(resp)

			_, _ = w.Write(respBytes)

			return
		} else if r.Method == "PUT" {
			if r.URL.Query().Get("confirm") == "1" {
				if iss.Verified {
					w.WriteHeader(400)
					return
				}

				ok := s.DB.ValidateIssue(rp.ID, iss.ID, r.URL.Query().Get("p"))

				if !ok {
					w.WriteHeader(403)
					return
				}

				go func(rp Report, com Issue) {
					s.ES.SendIssueMail(rp, com)
				}(*rp, *iss)

				return
			}

			return
		}
	}).Methods("GET", "POST", "PUT")

	r.HandleFunc("/report/{id}/{version}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		id := vars["id"]
		if id == "" {
			w.WriteHeader(404)
			return
		}

		verStr := vars["version"]
		ver, err := strconv.Atoi(verStr)
		if err != nil || ver <= 0 {
			w.WriteHeader(404)
			return
		}

		if r.Method == "GET" {
			s.serveRevision(w, id, ver)
			return
		}
	}).Methods("GET")

	r.HandleFunc("/report/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		id := vars["id"]
		if id == "" {
			w.WriteHeader(404)
			return
		}

		if r.Method == "GET" {
			rp := s.DB.GetReport(id)
			if rp == nil {
				w.WriteHeader(404)
				return
			}

			s.serveRevision(w, id, rp.Revisions)
			return
		} else if r.Method == "PUT" {
			reqBytes, _ := ioutil.ReadAll(r.Body)

			req := CreateRevisionRequest{}

			json.Unmarshal(reqBytes, &req)

			rp := s.DB.GetReport(id)

			if req.Email == "" {
				req.Email = rp.Email
			}

			rev := s.DB.CreateRevision(rp.ID, req.Email, req.Answers, req.Password, req.Public)
			if rev == nil {
				w.WriteHeader(401)
				return
			}

			go func(rp Report, rev Revision) {
				s.ES.SendRevisionMail(rp, rev)
			}(*rp, *rev)

			resp := CreateRevisionResponse{
				ID:       rp.ID,
				Password: rp.Token,
				Version:  rev.Version,
			}

			respBytes, _ := json.Marshal(resp)

			_, _ = w.Write(respBytes)

			return
		}
	}).Methods("GET", "PUT")

	r.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			reqBytes, _ := ioutil.ReadAll(r.Body)

			req := CreateReportRequest{}

			json.Unmarshal(reqBytes, &req)

			rp := s.DB.CreateReport(req.Email, req.Public)
			if rp == nil {
				w.WriteHeader(500)
				return
			}

			rev := s.DB.CreateRevision(rp.ID, req.Email, req.Answers, rp.Token, req.Public)
			if rev == nil {
				w.WriteHeader(500)
				return
			}

			go func(rp Report) {
				s.ES.SendReportMail(rp)
			}(*rp)

			resp := CreateRevisionResponse{
				ID:       rp.ID,
				Password: rp.Token,
				Version:  rev.Version,
			}

			respBytes, _ := json.Marshal(resp)

			_, _ = w.Write(respBytes)
		}
	}).Methods("POST")

	r.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			fileBytes, _ := ioutil.ReadAll(r.Body)

			fileName := s.DB.UploadDocument(fileBytes)

			respBytes, _ := json.Marshal(UploadFileResponse{File: fileName})

			w.Write(respBytes)
		}
	}).Methods("POST")

	r.HandleFunc("/documents/{document}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			vars := mux.Vars(r)
			fileName := vars["document"]
			fileBytes := s.DB.ReadDocument(fileName)
			if fileBytes == nil {
				w.WriteHeader(404)
				w.Write([]byte("not found"))
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.Write(fileBytes)
		}
	}).Methods("GET")

	r.HandleFunc("/keywords", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			keywords := s.DB.GetKeywords()

			if r.URL.Query().Get("format") == "list" {
				kwResp := KeywordListResponse{}

				for _, k := range keywords {
					kwResp = append(kwResp, k.keyword)
				}

				kwBytes, _ := json.Marshal(kwResp)

				w.Write(kwBytes)
			} else {
				kwResp := KeywordsResponse{}

				for _, k := range keywords {
					kwResp.Keywords = append(kwResp.Keywords, Keyword{
						Keyword: k.keyword,
						Count:   len(k.reports),
					})
				}

				kwBytes, _ := json.Marshal(kwResp)

				w.Write(kwBytes)
			}
		}
	}).Methods("GET")

	r.HandleFunc("/keywords/{keyword}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		kw := vars["keyword"]
		if kw == "" {
			w.WriteHeader(404)
			return
		}

		if r.Method == "GET" {
			keyword := s.DB.GetKeyword(kw)

			kwResp := KeywordResponse{
				Count:   len(keyword.reports),
				Keyword: keyword.keyword,
			}

			for _, k := range keyword.reports {
				r := s.DB.LatestRevision(k)

				title := ExtractField(s.DB.questions, r.Answers, []string{"MD", "1"})

				authors := ExtractFields(s.DB.questions, r.Answers, []string{"MD", "6", "*", "1"})

				cc := s.DB.GetReportIssues(r.ReportID, false)
				comments := 0
				for range cc {
					comments++
				}

				kwResp.Results = append(kwResp.Results, Result{
					ID:        r.ReportID,
					Title:     title,
					Authors:   authors,
					UpdatedAt: r.CreatedAt,
					Revisions: r.Version,
					Issues:    comments,
				})
			}

			kwBytes, _ := json.Marshal(kwResp)

			w.Write(kwBytes)
		}
	}).Methods("GET")

	r.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			offset, _ := strconv.Atoi(r.URL.Query().Get("o"))
			limit, _ := strconv.Atoi(r.URL.Query().Get("l"))
			if offset < 0 {
				offset = 0
			}
			if limit < 0 {
				limit = 0
			}
			if limit > 100 {
				limit = 100
			}

			originalQuery := r.URL.Query().Get("q")
			originalCategory := r.URL.Query().Get("c")
			originalKeyword := r.URL.Query().Get("k")

			fields := r.URL.Query().Get("f")

			sections := strings.Split(fields, ",")

			rc := make(chan *Revision)
			if originalCategory != "" || originalKeyword != "" {
				go func(rc chan *Revision, k1 string, k2 string) {
					defer close(rc)

					var rps1 map[string]bool

					if k1 != "" {
						rps1 = map[string]bool{}
						cat := s.DB.GetCategory(k1)
						if cat == nil {
							return
						}
						for _, repID := range cat.reports {
							rps1[repID] = true
						}
					}

					if k2 != "" {
						var rps2 map[string]bool
						kws := strings.Split(k2, ",")
						for _, k := range kws {
							rps2 = map[string]bool{}
							kw := s.DB.GetKeyword(k)
							for _, repID := range kw.reports {
								if rps1 == nil || rps1[repID] {
									rps2[repID] = true
								}
							}
							rps1 = rps2
						}
					}

					for repID := range rps1 {
						rc <- s.DB.LatestRevision(repID)
					}
				}(rc, originalCategory, originalKeyword)
			} else {
				rc = s.DB.GetLatestRevisions(false)
			}

			query := strings.ToLower(originalQuery)

			revisions := []*Revision{}
			for r := range rc {
				found := false
				if query == "" {
					found = true
				} else {
					if strings.Contains(r.ReportID, originalQuery) {
						found = true
					} else {
						for _, sec := range sections {
							txt := strings.ToLower(ExtractSectionText(s.DB.questions, r.Answers, sec))
							if strings.Contains(txt, query) {
								found = true
								break
							}
						}
					}
				}

				if found {
					revisions = append(revisions, r)
				}
			}

			count := len(revisions)

			sort.Slice(revisions, func(i, j int) bool {
				return revisions[i].CreatedAt.After(revisions[j].CreatedAt)
			})

			i := 0
			results := []Result{}
			for _, r := range revisions {
				if i >= offset && len(results) < limit {
					title := ExtractField(s.DB.questions, r.Answers, []string{"MD", "1"})

					authors := ExtractFields(s.DB.questions, r.Answers, []string{"MD", "6", "*", "1"})

					cc := s.DB.GetReportIssues(r.ReportID, false)
					comments := 0
					for range cc {
						comments++
					}

					results = append(results, Result{
						ID:        r.ReportID,
						Title:     title,
						Authors:   authors,
						UpdatedAt: r.CreatedAt,
						Revisions: r.Version,
						Issues:    comments,
					})
				}
				i++
			}

			searchBytes, _ := json.Marshal(SearchResponse{
				Count:   count,
				Results: results,
				Query:   originalQuery,
			})

			w.Write(searchBytes)
		}
	}).Methods("GET")

	r.HandleFunc("/contribute", func(w http.ResponseWriter, r *http.Request) {
		survey := struct {
			Answers json.RawMessage `json:"answers"`
			ReToken string          `json:"reToken"`
		}{}

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		err = json.Unmarshal(b, &survey)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		secret := "<GOOGLE CAPTCHE SECRET>"
		resp, err := http.Get("https://www.google.com/recaptcha/api/siteverify?secret=" + secret + "&response=" + survey.ReToken)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		reResponse := struct {
			Success bool `json:"success"`
		}{}

		reBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		err = json.Unmarshal(reBytes, &reResponse)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		if !reResponse.Success {
			w.WriteHeader(401)
			_, _ = w.Write([]byte("\"Incorrect captcha.\""))
			return
		}

		contr := struct {
			CreatedAt time.Time       `json:"createdAt"`
			Answers   json.RawMessage `json:"answers"`
		}{
			CreatedAt: time.Now(),
			Answers:   survey.Answers,
		}

		contrBytes, _ := json.Marshal(contr)

		s.DB.AddContribution(contrBytes)

		txt := string(survey.Answers)

		err = s.ES.SendMail("survey@aime-registry.org", "Survey participation", txt)
		if err != nil {
			w.WriteHeader(500)
			_, _ = w.Write([]byte("\"Error sending email.\""))
			return
		}

		_, _ = w.Write([]byte("\"OK\""))
	})

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	})

	s.srv = &http.Server{
		Addr:         "0.0.0.0:" + strconv.Itoa(s.Port),
		Handler:      c.Handler(r),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	s.srv.ListenAndServe()
}

func (s *Server) Shutdown() {
	s.srv.Shutdown(context.TODO())
}
