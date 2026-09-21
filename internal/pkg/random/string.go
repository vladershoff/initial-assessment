package random

import (
	"math/rand/v2"
	"strings"
)

// RandomString генерирует случайную строку заданной длины n
func RandomString(n int) string {
	// Набор символов, из которых будет состоять строка
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var sb strings.Builder
	sb.Grow(n) // Оптимизируем выделение памяти

	for i := 0; i < n; i++ {
		// rand.IntN выбирает случайный индекс из диапазона длины алфавита
		randomIndex := rand.IntN(len(alphabet))
		sb.WriteByte(alphabet[randomIndex])
	}

	return sb.String()
}
