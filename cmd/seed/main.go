// Command seed creates the initial admin user and sample news posts.
// Run after the first database migration:
//
//	go run ./cmd/seed
package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"argowil/backend/config"
	"argowil/backend/internal/database"
	"argowil/backend/internal/user"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	repo := user.NewRepository(db)
	svc := user.NewService(repo)

	ctx := context.Background()

	u, err := repo.FindByEmail(ctx, "stijn.jakobs@ordnary.com")
	if err != nil {
		u, err = svc.Create(ctx, user.CreateUserRequest{
			FirstName: "Stijn",
			LastName:  "Jakobs",
			Email:     "stijn.jakobs@ordnary.com",
			Password:  "1234",
			Role:      "admin",
		})
		if err != nil {
			log.Fatalf("create user: %v", err)
		}
		log.Printf("admin user created - id: %d  email: %s", u.ID, u.Email)
	} else {
		log.Printf("admin user already exists - id: %d  email: %s", u.ID, u.Email)
	}

	type seedPost struct {
		Title     string
		Body      string
		ImageURL  string
		CreatedAt time.Time
	}

	posts := []seedPost{
		{
			Title:     "Nieuwe parkeerplaatsen achter het magazijn",
			Body:      "Vanaf maandag zijn de extra parkeerplaatsen achter het magazijn open. Zo blijft het voorste plein vrij voor bezoekers en leveranciers.",
			ImageURL:  "asset://arg-094-small.jpg",
			CreatedAt: time.Date(2026, time.May, 26, 8, 15, 0, 0, time.UTC),
		},
		{
			Title:     "Koffiehoek op de eerste verdieping vernieuwd",
			Body:      "De koffiehoek bij kantoor is opgefrist en weer in gebruik. Kleine verandering, maar wel fijner voor de korte pauzes tussendoor.",
			ImageURL:  "asset://arg-086-small.jpg",
			CreatedAt: time.Date(2026, time.May, 22, 9, 5, 0, 0, time.UTC),
		},
		{
			Title:     "Team buitendienst start eerder bij warm weer",
			Body:      "Bij hogere temperaturen starten we deze week waar mogelijk wat eerder. Dat werkt prettiger buiten en helpt om het werk gelijkmatiger over de dag te verdelen.",
			ImageURL:  "asset://arg-085-small.jpg",
			CreatedAt: time.Date(2026, time.May, 19, 6, 45, 0, 0, time.UTC),
		},
		{
			Title:     "Nieuwe bedrijfskleding wordt per team uitgedeeld",
			Body:      "De eerste levering van de nieuwe bedrijfskleding is binnen. Teamleiders laten deze week per ploeg weten wanneer jullie alles kunnen ophalen.",
			ImageURL:  "asset://Groepsfoto-1.jpg",
			CreatedAt: time.Date(2026, time.May, 15, 14, 20, 0, 0, time.UTC),
		},
		{
			Title:     "Materialenpunt bij hal B is verplaatst",
			Body:      "Het materialenpunt zit nu naast de zijdeur van hal B. Daardoor kunnen de ochtendploegen sneller starten zonder eerst om te lopen.",
			ImageURL:  "asset://arg-001-small.jpg",
			CreatedAt: time.Date(2026, time.May, 12, 7, 30, 0, 0, time.UTC),
		},
	}

	createdPosts := 0
	for _, post := range posts {
		var existingID uint
		err := db.GetContext(ctx, &existingID, `SELECT id FROM news_posts WHERE title = ? LIMIT 1`, post.Title)
		if err == nil {
			continue
		}
		if err != sql.ErrNoRows {
			log.Fatalf("check news post %q: %v", post.Title, err)
		}

		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO news_posts (title, body, image_url, author_id, created_at) VALUES (?, ?, ?, ?, ?)`,
			post.Title, post.Body, post.ImageURL, u.ID, post.CreatedAt,
		); err != nil {
			log.Fatalf("create news post %q: %v", post.Title, err)
		}
		createdPosts++
	}

	log.Printf("news seed complete - created %d of %d sample posts", createdPosts, len(posts))
}
