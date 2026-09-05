package server

import (
	"net/http"

	"github.com/nkanaev/yarr/src/storage"
)

type StorageProvider interface {
	GetStorage(r *http.Request) storage.Storage
}

type LocalStorage struct {
	storage storage.Storage
}

func NewLocalStorage(db storage.Storage) *LocalStorage {
	return &LocalStorage{storage: db}
}

func (s *LocalStorage) GetStorage(r *http.Request) storage.Storage {
	return s.storage
}

func (s *Server) db(r *http.Request) storage.Storage {
	return s.Storage.GetStorage(r)
}
