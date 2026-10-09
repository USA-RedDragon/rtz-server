package v1dot4_test

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/iotest"

	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/USA-RedDragon/rtz-server/internal/logparser"
	v1dot4 "github.com/USA-RedDragon/rtz-server/internal/server/controllers/v1.4"
	"github.com/USA-RedDragon/rtz-server/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type memFile struct {
	*bytes.Reader
	buf    *bytes.Buffer
	closed *int
	mu     *sync.Mutex
}

func (f memFile) Write(p []byte) (int, error) { return f.buf.Write(p) }

func (f memFile) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	*f.closed++
	return nil
}

type memStorage struct {
	mu     sync.Mutex
	files  map[string]*bytes.Buffer
	opened int
	closed int
}

func (s *memStorage) file(buf *bytes.Buffer) memFile {
	s.opened++
	return memFile{Reader: bytes.NewReader(buf.Bytes()), buf: buf, closed: &s.closed, mu: &s.mu}
}

func (s *memStorage) Open(name string) (storage.File, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	buf, ok := s.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return s.file(buf), nil
}

func (s *memStorage) Create(name string) (storage.File, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	buf := &bytes.Buffer{}
	s.files[name] = buf
	return s.file(buf), nil
}

func (s *memStorage) Mkdir(string, fs.FileMode) error     { return nil }
func (s *memStorage) MkdirAll(string, fs.FileMode) error  { return nil }
func (s *memStorage) Remove(string) error                 { return nil }
func (s *memStorage) Sub(string) (storage.Storage, error) { return s, nil }
func (s *memStorage) Close() error                        { return nil }

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Device{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Device{DongleID: "abc", Serial: "serial", PublicKey: "key"}).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func putUpload(t *testing.T, db *gorm.DB, store *memStorage, path string, body io.Reader) int {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/v1.4/abc/upload?path="+path, body)
	c.Params = gin.Params{{Key: "dongle_id", Value: "abc"}}
	c.Set("db", db)
	c.Set("storage", storage.Storage(store))
	c.Set("logQueue", &logparser.LogQueue{})
	v1dot4.PUTUpload(c)
	return w.Code
}

func TestPUTUploadClosesInvalidQlog(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	for _, path := range []string{"0000001a--abcdef0123--0/qlog.bz2", "0000001a--abcdef0123--0/qlog.zst"} {
		store := &memStorage{files: map[string]*bytes.Buffer{}}
		if code := putUpload(t, db, store, path, strings.NewReader("not a qlog")); code != http.StatusBadRequest {
			t.Errorf("%s: got status %d, want %d", path, code, http.StatusBadRequest)
		}
		if store.opened != store.closed {
			t.Errorf("%s: opened %d files but closed %d", path, store.opened, store.closed)
		}
	}
}

func TestPUTUploadClosesFailedWrite(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	store := &memStorage{files: map[string]*bytes.Buffer{}}
	if code := putUpload(t, db, store, "0000001a--abcdef0123--0/qlog.bz2", iotest.ErrReader(errors.New("read failed"))); code != http.StatusInternalServerError {
		t.Errorf("got status %d, want %d", code, http.StatusInternalServerError)
	}
	if store.opened != store.closed {
		t.Errorf("opened %d files but closed %d", store.opened, store.closed)
	}
}
