package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/forrest-bajbek/bombs/types"
)

func (s *Service) CreateMessage(requestingUserID int, chatID int, text string, files []types.NewFile) (int, error) {
	if strings.TrimSpace(text) == "" && len(files) == 0 {
		return -1, errors.New("Message must contain text or at least one photo.")
	}
	if len(text) > 1024 {
		return -1, errors.New("Message must be less than 1024 characters.")
	}
	if len(files) > types.MaxFilesPerMessage {
		return -1, fmt.Errorf("You can attach at most %d photos.", types.MaxFilesPerMessage)
	}

	var uploadBytes int64
	for _, f := range files {
		if len(f.Content) == 0 {
			return -1, errors.New("Empty file.")
		}
		if len(f.Content) > types.MaxFileBytes {
			return -1, fmt.Errorf("Photos must be smaller than %dMB.", types.MaxFileBytes>>20)
		}
		if !types.AllowedImageMimeTypes[f.MimeType] {
			return -1, errors.New("Only JPEG, PNG, GIF and WebP images are allowed.")
		}
		uploadBytes += int64(len(f.Content))
	}

	if uploadBytes > 0 {
		used, err := s.StorageUsedBytes()
		if err != nil {
			return -1, err
		}
		if used+uploadBytes > types.StorageBudgetBytes {
			return -1, errors.New("Photo storage is full. Bomb a chat to free space.")
		}
	}

	return s.repo.CreateMessage(requestingUserID, chatID, text, files)
}

func (s *Service) GetMessageByID(requestingUserID int, chatID int, messageID int) (*types.ChannelMessage, error) {
	return s.repo.GetMessageByID(requestingUserID, chatID, messageID)
}

func (s *Service) GetMessagesByChatID(requestingUserID int, chatID int) (*[]types.ChannelMessage, error) {
	return s.repo.GetMessagesByChatID(requestingUserID, chatID)
}

func (s *Service) DeleteMessageByUserID(userID int) error {
	return s.repo.DeleteMessageByUserID(userID)
}
