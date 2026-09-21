package domain

import "time"

// Link представляет собой модель данных ссылки.
type Link struct {
	ShortCode   string
	OriginalUrl string
	CreatedAt   time.Time
	Visits      int
}
