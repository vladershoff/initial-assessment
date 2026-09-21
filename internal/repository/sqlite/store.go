package sqlite

import (
	"database/sql"
	"fmt"
)

// SqliteStore представляет собой хранилище данных на базе SQLite.
//
// Структура реализует интерфейс хранилища для работы со ссылками, инкапсулируя
// в себе подключение к базе данных и предоставляя методы для выполнения
// операций CRUD и счетчиков.
type SqliteStore struct {
	// db — подключение к базе данных SQLite.
	db *sql.DB
}

// InitStore инициализирует подключение к базе данных SQLite и создает
// необходимую структуру таблиц, если они еще не созданы.
//
// Функция открывает файл базы данных "assessment.db", создает таблицу 'links'
// и возвращает готовый к работе объект хранилища.
//
// Возвращает:
//   - *SqliteStore: указатель на инициализированное хранилище при успешном запуске.
//   - error: обернутую ошибку, если не удалось открыть файл БД или создать таблицу.
func InitStore() (*SqliteStore, error) {
	// Открываем/создаем БД
	//
	// Параметры:
	//   - _journal_mode=WAL: включает конкурентное чтение/запись
	//   - _busy_timeout=5000: заставляет горутину ждать до 5000 мс, если база занята
	// другим запросом, вместо выброса ошибки
	db, err := sql.Open("sqlite", "assessment.db?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("Ошибка при открытии/создании БД (InitStore): %w", err)
	}

	// Создаем таблицу ссылок
	query := `
		CREATE TABLE IF NOT EXISTS links (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			short_code TEXT NOT NULL,
			original_url TEXT NOT NULL UNIQUE,
			created_at DATETIME NOT NULL DEFAULT current_timestamp,
			visits INTEGER NOT NULL DEFAULT 0
		);`
	_, err = db.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("Ошибка при создании таблицы links: %w", err)
	}

	// Формируем структуру ответа
	store := SqliteStore{
		db: db,
	}

	return &store, err
}

// Close безопасно закрывает соединение с базой данных SQLite. Метод проверяет
// инициализацию объекта базы данных, поэтому его вызов безопасен даже в том
// случае, если соединение не было открыто или равно nil.
func (store *SqliteStore) Close() {
	if store.db != nil {
		store.db.Close()
	}
}
