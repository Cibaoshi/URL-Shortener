package storage

import (
	"sync"
	"time"

	"shortenergo/internal/utils"
)

type ClickInfo struct {
	Time time.Time `json:"time"`
	IP   string    `json:"ip"`
	UA   string    `json:"ua"`
}

type UrlStats struct {
	URL       string      `json:"url"`
	Clicks    int         `json:"clicks"`
	LastClick time.Time   `json:"last_click"`
	History   []ClickInfo `json:"history"`
}

type Store struct {
	data map[string]*UrlStats
	mu   sync.RWMutex
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]*UrlStats),
	}
}

func (s *Store) Save(originalURL string) string {
	for {
		code := utils.GenerateShortCode()
		s.mu.Lock()
		if _, exists := s.data[code]; !exists {
			s.data[code] = &UrlStats{
				URL:     originalURL,
				History: make([]ClickInfo, 0),
			}
			s.mu.Unlock()
			return code
		}
		s.mu.Unlock()
	}
}

func (s *Store) Get(code string) (*UrlStats, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats, ok := s.data[code]
	return stats, ok
}

func (s *Store) AddClick(code, ip, ua string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if stats, ok := s.data[code]; ok {
		stats.Clicks++
		stats.LastClick = time.Now()
		stats.History = append(stats.History, ClickInfo{
			Time: stats.LastClick,
			IP:   ip,
			UA:   ua,
		})
		if len(stats.History) > 100 {
			stats.History = stats.History[len(stats.History)-100:]
		}
	}
}

func (s *Store) GetAll() map[string]*UrlStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copyData := make(map[string]*UrlStats)
	for k, v := range s.data {
		copyData[k] = v
	}
	return copyData
}
