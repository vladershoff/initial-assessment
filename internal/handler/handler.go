package handler

import (
	"initial-assessment/internal/domain"

	"github.com/gorilla/mux"
)

type LinkService interface {
	InsertLink(url string) (*domain.Link, error)
	VisitLink(shortCode string) (*domain.Link, error)
	GetLinksList(offset int, limit int) (*[]domain.Link, *int, error)
	GetLink(shortCode string) (*domain.Link, error)
	DeleteLink(shortCode string) error
}

type CacheService interface {
	Get(key string) (any, bool)
	Set(key string, value any)
	Remove(key string)
}

func NewRouter(store LinkService, cache CacheService) *mux.Router {
	// Создаем роутер
	router := mux.NewRouter()

	// Инициализируем API ссылок
	linkHandler := &LinkHandler{
		Store: store,
		Cache: cache,
	}

	// Создание короткой ссылки
	router.HandleFunc("/links", linkHandler.Create).Methods("post")

	// Получение оригинальной ссылки
	router.HandleFunc("/links/{shortCode}", linkHandler.Get).Methods("get")

	// Получение списка ссылок
	router.HandleFunc("/links", linkHandler.GetList).Methods("get")

	// Удаление ссылки
	router.HandleFunc("/links/{shortCode}", linkHandler.Remove).Methods("delete")

	// Статистика
	router.HandleFunc("/links/{shortCode}/stats", linkHandler.Stats).Methods("get")

	return router
}
