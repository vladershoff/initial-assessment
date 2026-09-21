package handler

import (
	"encoding/json"
	"errors"
	"initial-assessment/internal/domain"
	"initial-assessment/internal/pkg/httputil"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type LinkHandler struct {
	Store LinkService
	Cache CacheService
}

// Create обрабатывает POST-запросы на создание короткой ссылки.
//
// Метод ожидает в теле запроса JSON вида: {"url": "https://example.com"}.
// В случае успеха возвращает JSON со сгенерированным кодом: {"short_code": "xYz12"}.
//
// Возможные HTTP-ответы:
//   - 200 OK: Ссылка успешно создана и сохранена.
//   - 400 Bad Request: Передан некорректный JSON.
//   - 500 Internal Server Error: Ошибка при работе с базой данных.
func (h *LinkHandler) Create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var (
		err  error
		link *domain.Link
		body struct {
			Url string `json:"url"`
		}
	)

	// Парсим json в теле запроса
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	// Сохраняем Url в БД и получаем корткий код
	link, err = h.Store.InsertLink(body.Url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Формируем структуру ответа
	var resp = struct {
		ShortCode string `json:"short_code"`
	}{ShortCode: link.ShortCode}

	// Формируем json и кладем его в ответ для клиента
	httputil.ResponseJSON(w, 200, resp)
}

// Get обрабатывает GET-запросы для получения информации о ссылке по ее короткому коду.
//
// Метод извлекает 'shortCode' из URL, фиксирует посещение в базе данных,
// обновляет данные в кэше и возвращает оригинальный URL со счетчиком просмотров.
//
// Возможные HTTP-ответы:
//   - 200 OK: Информация успешно найдена и возвращена в JSON.
//   - 404 Not Found: Указанный короткий код не существует в системе.
//   - 500 Internal Server Error: Внутренняя ошибка при работе с базой данных.
func (h *LinkHandler) Get(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var (
		err  error
		link *domain.Link
	)

	// Извлекаем короткую ссылку из запроса
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	// Ищем короткую ссылку в БД и увеличиваем количество просмотров
	link, err = h.Store.VisitLink(shortCode)

	// Если в БД не найдена ссылка возвращаем статус 404
	if errors.Is(err, domain.ErrShortCodeNotDefined) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Для других ошибок возвращаем статус 500
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Кэшируем данные
	h.Cache.Set(link.ShortCode, *link)

	// Формируем структуру ответа
	var resp = struct {
		Url    string `json:"url"`
		Visits int    `json:"visits"`
	}{
		Url:    link.OriginalUrl,
		Visits: link.Visits,
	}

	// Формируем json и кладем его в ответ для клиента
	httputil.ResponseJSON(w, 200, resp)
}

// GetList обрабатывает GET-запросы для получения пагинированного списка ссылок.
//
// Метод ожидает query-параметры 'limit' и 'offset' для управления пагинацией.
// Возвращает массив элементов с оригинальными URL и короткими кодами, а также общее число записей.
//
// Возможные HTTP-ответы:
//   - 200 OK: Список успешно сформирован и передан в JSON.
//   - 500 Internal Server Error: Ошибка валидации параметров или сбой при чтении из БД.
func (h *LinkHandler) GetList(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// Извлекаем параметры из строки запроса
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		http.Error(w, "Неверно указан параметр offset", http.StatusInternalServerError)
		return
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		http.Error(w, "Неверно указан параметр limit", http.StatusInternalServerError)
		return
	}

	// Извлекаем из БД список ссылок
	links, allCount, err := h.Store.GetLinksList(offset, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Структура для json ответа на клиент
	type LinksListElem struct {
		Url       string `json:"url"`
		ShortCode string `json:"short_code"`
	}
	var linkPage = struct {
		AllCount int             `json:"all_count"`
		Links    []LinksListElem `json:"links"`
	}{
		AllCount: *allCount,
		Links:    make([]LinksListElem, 0, len(*links)),
	}

	// Перекладываем данные из БД в структуру ответа на клиент
	for _, v := range *links {
		linkElem := LinksListElem{
			Url:       v.OriginalUrl,
			ShortCode: v.ShortCode,
		}
		linkPage.Links = append(linkPage.Links, linkElem)
	}

	// Формируем json и кладем его в ответ для клиента
	httputil.ResponseJSON(w, 200, linkPage)
}

// Remove обрабатывает DELETE-запросы для удаления короткой ссылки по ее коду.
//
// Метод извлекает 'shortCode' из URL, удаляет запись из базы данных,
// а также удаляет соответствующее значение в кэше.
//
// Возможные HTTP-ответы:
//   - 200 OK: Ссылка успешно удалена (тело ответа пустое).
//   - 404 Not Found: Ссылка с указанным коротким кодом не существует.
//   - 500 Internal Server Error: Внутренняя ошибка при попытке удаления из БД.
func (h *LinkHandler) Remove(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	err := h.Store.DeleteLink(shortCode)
	if errors.Is(err, domain.ErrShortCodeNotDefined) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Удаляем ссылку из кэша
	h.Cache.Remove(shortCode)
}

// Stats обрабатывает GET-запросы для получения расширенной статистики короткой ссылки.
//
// Метод сначала ищет информацию в кэше. Если её там нет, запрашивает данные из БД
// и кэширует результат. Возвращает оригинальный URL, код, количество просмотров и дату создания.
//
// Возможные HTTP-ответы:
//   - 200 OK: Статистика успешно найдена и возвращена в JSON.
//   - 404 Not Found: Ссылка с указанным коротким кодом не существует.
//   - 500 Internal Server Error: Внутренняя ошибка при работе с БД или кэшем.
func (h *LinkHandler) Stats(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// Извлекаем короткий код ссылки из запроса
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	var link domain.Link

	// Ищем данные в кэше
	cacheLink, cacheOk := h.Cache.Get(shortCode)
	if cacheOk {
		// Извлеченное значение пытаемся привести к типу Link
		var linkOk bool
		link, linkOk = cacheLink.(domain.Link)
		if !linkOk {
			// По запрошенному shortCode хранится не объект Link,
			// возвращаем ошибку
			http.Error(w, "Некорректный короткий код ссылки", http.StatusInternalServerError)
			return
		}
	} else {
		// В кэше не найдена ссылка по shortCode, ищем в БД
		ptrLink, storeErr := h.Store.GetLink(shortCode)

		// В БД не найден shortCode возвращаем статус 404
		if errors.Is(storeErr, domain.ErrShortCodeNotDefined) {
			http.Error(w, storeErr.Error(), http.StatusNotFound)
			return
		}

		// При других ошибках БД возвращаем статус 500
		if storeErr != nil {
			http.Error(w, storeErr.Error(), http.StatusInternalServerError)
			return
		}

		// Если ошибок нет извлекаем ссылку из указателя и кэшируем
		link = *ptrLink
		h.Cache.Set(shortCode, link)
	}

	// Формируем структуру ответа
	var resp = struct {
		ShortCode string    `json:"short_code"`
		Url       string    `json:"url"`
		Visits    int       `json:"visits"`
		CreatedAt time.Time `json:"create_at"`
	}{
		ShortCode: link.ShortCode,
		Url:       link.OriginalUrl,
		Visits:    link.Visits,
		CreatedAt: link.CreatedAt,
	}

	// Формируем json и кладем его в ответ для клиента
	httputil.ResponseJSON(w, 200, resp)
}
