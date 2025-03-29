package handlers

import "sync"

const (
	AddUserState             = "add_user"
	AddChatState             = "add_chat"
	RemoveUserState          = "remove_user"
	RemoveChatState          = "remove_chat"
	ListUsersState           = "list_users"
	ListChatsState           = "list_chats"
	SendNotificationState    = "send_message"
	ConfirmNotificationState = "confirm_notification"
	MenuState                = "menu"
)

type State struct {
	s  map[int64]string
	mu sync.RWMutex
}

func NewState() *State {
	return &State{
		s:  make(map[int64]string),
		mu: sync.RWMutex{},
	}
}

func (s *State) Set(chatID int64, state string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.s[chatID] = state
}

func (s *State) Get(chatID int64) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.s[chatID]
}

func (s *State) Delete(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.s, chatID)
}
