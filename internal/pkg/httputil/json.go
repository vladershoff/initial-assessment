package httputil

import (
	"encoding/json"
	"net/http"
)

func ResponseJSON(w http.ResponseWriter, status int, data any) {
	// Сериализуем данные в память ДО изменения заголовков ResponseWriter
	payload, err := json.Marshal(data)
	if err != nil {
		// Если произошла ошибка, заголовок application/json еще не установлен.
		// Будет корректно установлен заголовок ошибки и статус:
		// Content-Type: text/plain и статус 500.
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Когда json сформирован устанавливаем для него заголовок
	w.Header().Set("Content-Type", "application/json")

	// Устанавливаем необходимый статус
	w.WriteHeader(status)

	// Записываем ответ
	w.Write(payload)
}
