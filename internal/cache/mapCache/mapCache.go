package mapcache

import "sync"

type MapCache struct {
	cache map[string]any
	mtx   sync.RWMutex
}

// Init инициализирует и возвращает новый экземпляр MapCache.
//
// Возвращает:
// - *MapCache: указатель на созданный пустой кеш.
func Init() *MapCache {
	return &MapCache{
		cache: map[string]any{},
	}
}

// Get возвращает значение по указанному ключу из кеша.
// Метод безопасен для конкурентного использования горутинами.
//
// Параметры:
// - key string: уникальный ключ для поиска в кеше.
//
// Возвращает:
// - any: сохраненное значение (nil, если ключ не найден).
// - bool: флаг успешного поиска (true, если ключ существует).
func (mapcache *MapCache) Get(key string) (any, bool) {
	mapcache.mtx.RLock()
	defer mapcache.mtx.RUnlock()
	value, ok := mapcache.cache[key]
	return value, ok
}

// Set сохраняет или обновляет значение для заданного ключа.
// Блокирует кеш на запись, предотвращая состояние гонки.
//
// Параметры:
// - key string: ключ, под которым будет сохранено значение.
// - value any: любые данные для сохранения.
func (mapcache *MapCache) Set(key string, value any) {
	mapcache.mtx.Lock()
	defer mapcache.mtx.Unlock()
	mapcache.cache[key] = value
}

// Remove удаляет запись из кеша по её ключу.
// Если ключа не существует, операция завершается без ошибок.
//
// Параметры:
// - key string: ключ элемента, который нужно удалить.
func (mapcache *MapCache) Remove(key string) {
	mapcache.mtx.Lock()
	defer mapcache.mtx.Unlock()
	delete(mapcache.cache, key)
}
