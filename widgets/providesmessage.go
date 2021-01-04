package widgets

import (
	"gitea.com/iwakuramarie/nyat/lib"
	"gitea.com/iwakuramarie/nyat/lib/ui"
	"gitea.com/iwakuramarie/nyat/models"
)

type PartInfo struct {
	Index []int
	Msg   *models.MessageInfo
	Part  *models.BodyStructure
}

type ProvidesMessage interface {
	ui.Drawable
	Store() *lib.MessageStore
	SelectedAccount() *AccountView
	SelectedMessage() (*models.MessageInfo, error)
	SelectedMessagePart() *PartInfo
}

type ProvidesMessages interface {
	ui.Drawable
	Store() *lib.MessageStore
	SelectedAccount() *AccountView
	SelectedMessage() (*models.MessageInfo, error)
	MarkedMessages() ([]uint32, error)
}
