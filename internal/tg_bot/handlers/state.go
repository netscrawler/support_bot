package handlers

import (
	"sync"
	"time"
)

const (
	addUserState    = "add_user"
	addChatState    = "add_chat"
	removeUserState = "remove_user"
	removeChatState = "remove_chat"

	sendNotificationState    = "send_message"
	confirmNotificationState = "confirm_notification"

	menuState = "menu"

	loadReportState = "load_report"
)

type State struct {
	s           map[int64]string
	msg         map[int64]string
	timers      map[int64]*time.Timer // Храним таймеры для каждого чата
	mu          sync.RWMutex
	cleanUpTime time.Duration
}

func NewState(cleanUpTime time.Duration) *State {
	return &State{
		s:           make(map[int64]string),
		msg:         make(map[int64]string),
		timers:      make(map[int64]*time.Timer),
		mu:          sync.RWMutex{},
		cleanUpTime: cleanUpTime,
	}
}

func (s *State) set(chatID int64, state string) {
	s.mu.Lock()
	s.s[chatID] = state
	s.mu.Unlock()
	s.cleanUpAfter(chatID, s.cleanUpTime)
}

func (s *State) get(chatID int64) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.s[chatID]
}

func (s *State) setMsgData(chatID int64, msg string) {
	s.mu.Lock()
	s.msg[chatID] = msg
	s.mu.Unlock()
	s.cleanUpAfter(chatID, s.cleanUpTime)
}

func (s *State) getMsgData(chatID int64) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.msg[chatID]

	return data, ok
}

func (s *State) delete(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.s, chatID)
	delete(s.msg, chatID)

	if timer, exists := s.timers[chatID]; exists {
		timer.Stop()
		delete(s.timers, chatID)
	}
}

// Очистка с отменой старого таймера.
func (s *State) cleanUpAfter(chatID int64, d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Отменяем старый таймер, если он существует
	if oldTimer, exists := s.timers[chatID]; exists {
		oldTimer.Stop()
	}

	// Создаем новый таймер
	timer := time.AfterFunc(d, func() {
		s.mu.Lock()
		delete(s.s, chatID)
		delete(s.msg, chatID)
		delete(s.timers, chatID)
		s.mu.Unlock()
	})

	// Сохраняем таймер в map
	s.timers[chatID] = timer
}
