package main

import (
	"aime/pkg/aime"
	"log"
)

func main() {
	db := &aime.DB{
		KeywordGroups: aime.ReadKeywordGroups("./keyword-groups.yaml"),
		Dir:           "./db/",
	}
	db.Create("./questionnaire.yaml")

	es := aime.NewEmailSender("<EMAIL HOST>", 587, "<EMAIL USERNAME>", "<EMAIL PASSWORD>")
	if err := es.LoadTemplates("./templates/"); err != nil {
		log.Fatal(err)
	}

	srv := aime.Server{
		Port: 9000,
		DB:   db,
		ES:   es,
	}

	kl, cl := db.BuildKeywordList([]string{"MD", "5"}, []string{"P", "3", "1"})

	log.Printf("Found %d categories\n", cl)
	log.Printf("Found %d keywords\n", kl)

	srv.Start()
}
