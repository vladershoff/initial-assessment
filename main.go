package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"

	mapCache "initial-assessment/internal/cache/mapcache"
	"initial-assessment/internal/handler"
	sqliteStore "initial-assessment/internal/repository/sqlite"

	_ "modernc.org/sqlite"
)

func main() {
	// Открываем/создаем БД и создаем таблицу links
	store, err := sqliteStore.InitStore()

	// Ошибка при инициализации БД. Показываем ошибку, ждем Enter от
	// пользователя и завершаем приложение
	if err != nil {
		fmt.Println(err.Error())
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
		return
	}

	// При выходе из main закрываем store
	defer store.Close()

	// Инициализируем хранилище для кэша
	cache := mapCache.Init()

	// Хранилище и кэш передаем в роутер
	router := handler.NewRouter(store, cache)

	// Регистрируем роутер и запускаем сервер
	http.Handle("/", router)
	fmt.Println("Server is listening...")
	http.ListenAndServe(":8181", nil)
}
