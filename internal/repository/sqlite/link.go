package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"initial-assessment/internal/domain"
	"initial-assessment/internal/pkg/random"
)

// createShortCode генерирует случайную текстовую строку фиксированной
// длины (10 символов) для использования в качестве короткого кода ссылки.
//
// Возвращает:
//   - string: уникальный (в рамках одной генерации) короткий идентификатор.
func createShortCode() string {
	return random.RandomString(10)
}

// InsertLink сохраняет новый URL в базе данных с генерацией короткого кода
// либо возвращает уже существующую запись, если этот URL был добавлен ранее.
//
// Параметры:
//   - url: оригинальный URL.
//
// Возвращает:
//   - *Link: указатель на структуру с данными ссылки (новой или ранее созданной).
//   - error: обернутую ошибку при сбое поиска, генерации или вставки данных в БД.
func (store *SqliteStore) InsertLink(url string) (*domain.Link, error) {
	var (
		query string
		link  = domain.Link{}
		err   error
	)

	// Ищем url в бд
	query = `
		SELECT
			short_code,
			original_url,
			created_at,
			visits
		FROM
			links
		WHERE
			original_url = ?`
	err = store.db.QueryRow(query, url).Scan(
		&link.ShortCode,
		&link.OriginalUrl,
		&link.CreatedAt,
		&link.Visits,
	)

	// Если отсутствует - создаем короткий код и добавляем в бд
	if errors.Is(err, sql.ErrNoRows) {
		shortCode := createShortCode()
		query = `
			INSERT INTO
				links(short_code, original_url)
			VALUES
				(?, ?)
			RETURNING
				short_code, original_url, created_at, visits;`
		err = store.db.QueryRow(query, shortCode, url).Scan(
			&link.ShortCode,
			&link.OriginalUrl,
			&link.CreatedAt,
			&link.Visits,
		)
	}

	if err != nil {
		err = fmt.Errorf("ошибка сохранения url в БД (InsertLink): %w", err)
	}

	return &link, err
}

// VisitLink увеличивает счетчик посещений (visits) для ссылки с указанным shortCode
// и возвращает обновленные данные о ссылке.
//
// Если ссылка с таким shortCode не найдена в базе данных, метод возвращает
// специальную ошибку ErrShortCodeNotDefined.
//
// Возвращает:
//   - *Link: указатель на структуру с актуальными данными из БД в случае успеха.
//   - error: ErrShortCodeNotDefined, если код не найден, либо обернутую системную ошибку БД.
func (store *SqliteStore) VisitLink(shortCode string) (*domain.Link, error) {
	var link = domain.Link{}

	query := `
		UPDATE
			links
		SET
			visits = visits + 1
		WHERE
			short_code = ?
		RETURNING
			short_code, original_url, created_at, visits;`
	// В БД поля short_code, original_url, created_at, visits принимают
	// значения NOT NULL, поэтому при вызове Scan паника не произойдет
	err := store.db.QueryRow(query, shortCode).Scan(
		&link.ShortCode,
		&link.OriginalUrl,
		&link.CreatedAt,
		&link.Visits,
	)

	// Чтобы во внешнем коде отследить, что короткая ссылка не найдена -
	// возвращаем специальную ошибку
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrShortCodeNotDefined
	}

	if err != nil {
		err = fmt.Errorf("ошибка счетчика посещения (VisitLink): %w", err)
	}

	return &link, err
}

// GetLinksList возвращает постраничный список ссылок, отсортированный по дате создания
// в обратном порядке (сначала новые), и общее количество записей в базе данных.
//
// Если по заданным смещениям записей не найдено, метод вернет пустой срез и общее количество, равное 0.
//
// Параметры:
//   - offset: количество пропускаемых записей с начала выборки.
//   - limit: максимальное количество возвращаемых записей в текущем запросе.
//
// Возвращает:
//   - *[]Link: указатель на срез со ссылками (может быть пустым, но не nil при успехе).
//   - *int: указатель на общее количество записей в таблице, удовлетворяющих условию.
//   - error: обернутую ошибку при сбое выполнения запроса или сканирования строк.
func (store *SqliteStore) GetLinksList(offset int, limit int) (*[]domain.Link, *int, error) {
	query := `
		SELECT
			count(1) over () all_cnt,
			short_code,
			original_url,
			created_at,
			visits
		FROM
			links
		ORDER BY
			created_at DESC
		LIMIT ? OFFSET ?`
	rows, err := store.db.Query(query, limit, offset)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка выполнения запроса (GetLinksList): %w", err)
	}

	defer rows.Close()

	var allCount int
	links := []domain.Link{}
	for rows.Next() {
		link := domain.Link{}
		err = rows.Scan(
			&allCount,
			&link.ShortCode,
			&link.OriginalUrl,
			&link.CreatedAt,
			&link.Visits,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("ошибка чтения строк из БД (GetLinksList): %w", err)
		}
		links = append(links, link)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("ошибка сканирования из БД (GetLinksList): %w", err)
	}

	return &links, &allCount, nil
}

// GetLink возвращает информацию о ссылке по её короткому коду shortCode.
//
// Если запись с таким коротким кодом отсутствует в базе данных, метод возвращает
// специальную ошибку ErrShortCodeNotDefined.
//
// Возвращает:
//   - *Link: указатель на структуру с данными ссылки при успешном поиске.
//   - error: ErrShortCodeNotDefined, если ссылка не найдена, либо обернутую ошибку при сбое запроса к БД.
func (store *SqliteStore) GetLink(shortCode string) (*domain.Link, error) {
	link := domain.Link{}
	query := `
		SELECT
			original_url,
			short_code,
			created_at,
			visits
		FROM
			links
		WHERE
			short_code = ?`
	err := store.db.QueryRow(query, shortCode).Scan(
		&link.OriginalUrl,
		&link.ShortCode,
		&link.CreatedAt,
		&link.Visits,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrShortCodeNotDefined
	}

	if err != nil {
		return nil, fmt.Errorf("ошибка получения информации по ссылке (GetLink): %w", err)
	}

	return &link, nil
}

// DeleteLink удаляет ссылку с указанным shortCode из базы данных.
//
// Если в базе данных не было записи с таким коротким кодом (удалено 0 строк),
// метод возвращает специальную ошибку ErrShortCodeNotDefined.
//
// Возвращает:
//   - error: ErrShortCodeNotDefined, если ссылка не найдена, nil в случае успешного удаления,
//     либо обернутую ошибку при сбое выполнения запроса к БД.
func (store *SqliteStore) DeleteLink(shortCode string) error {
	query := "DELETE FROM links WHERE short_code = ?"
	res, err := store.db.Exec(query, shortCode)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка удаления ссылки из БД (DeleteLink): %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrShortCodeNotDefined
	}

	return nil
}
