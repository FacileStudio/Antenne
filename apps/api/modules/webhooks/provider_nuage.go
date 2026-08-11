package webhooks

import (
	"fmt"
	"strconv"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

type nuagePayload struct {
	EventType    string `json:"event_type"`
	OccurredAt   string `json:"occurred_at"`
	InstanceName string `json:"instance_name"`
	Actor        *struct {
		ID    int    `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"actor"`
	Data *struct {
		File *struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			MimeType string `json:"mime_type"`
			Size     int64  `json:"size"`
		} `json:"file"`
		Folder *struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"folder"`
		Share *struct {
			ID              int    `json:"id"`
			SharedWithEmail string `json:"shared_with_email"`
			Permission      string `json:"permission"`
		} `json:"share"`
	} `json:"data"`
}

var nuage = Definition{
	Type:         settings.ProviderNuage,
	Label:        "Nuage",
	Authenticate: signedWith("sha256="),
	Handle: func(body []byte, _ Context) (Result, error) {
		var payload nuagePayload
		if err := decode(body, &payload); err != nil {
			return Result{}, err
		}

		if payload.EventType == "" || payload.Actor == nil || payload.Data == nil {
			return Result{Description: "unknown nuage event"}, nil
		}

		name := "unknown"
		subjectID := ""
		switch {
		case payload.Data.File != nil:
			name = payload.Data.File.Name
			subjectID = strconv.Itoa(payload.Data.File.ID)
		case payload.Data.Folder != nil:
			name = payload.Data.Folder.Name
			subjectID = strconv.Itoa(payload.Data.Folder.ID)
		case payload.Data.Share != nil:
			subjectID = strconv.Itoa(payload.Data.Share.ID)
		}

		normalized := map[string]any{
			"eventType":    payload.EventType,
			"instanceName": payload.InstanceName,
			"actorId":      payload.Actor.ID,
			"actorEmail":   payload.Actor.Email,
			"actorName":    payload.Actor.Name,
		}
		if file := payload.Data.File; file != nil {
			normalized["fileId"] = file.ID
			normalized["fileName"] = file.Name
			normalized["fileMimeType"] = file.MimeType
			normalized["fileSize"] = file.Size
		}
		if folder := payload.Data.Folder; folder != nil {
			normalized["folderId"] = folder.ID
			normalized["folderName"] = folder.Name
		}
		if share := payload.Data.Share; share != nil {
			normalized["shareId"] = share.ID
			normalized["sharedWithEmail"] = share.SharedWithEmail
			normalized["sharePermission"] = share.Permission
		}

		return Result{
			Description: fmt.Sprintf("%s: %s", payload.EventType, name),
			MessageHTML: nuageMessage(payload),
			EventType:   "nuage." + payload.EventType,
			EventTitle:  name,
			OccurredAt:  payload.OccurredAt,
			Actor: &envelope.Actor{
				ID:    str(strconv.Itoa(payload.Actor.ID)),
				Name:  str(payload.Actor.Name),
				Email: str(payload.Actor.Email),
			},
			Subject: &envelope.Subject{
				Type: envelope.SubjectUnknown,
				ID:   str(subjectID),
				Name: str(name),
			},
			Normalized:  normalized,
			ContainsPII: true,
		}, nil
	},
}

func nuageMessage(payload nuagePayload) string {
	icon := nuageEmoji(payload.EventType)
	actor := escape(payload.Actor.Name)

	switch payload.EventType {
	case "file.uploaded", "file.updated", "file.deleted", "file.shared":
		return nuageFileMessage(payload, icon, actor)
	case "folder.created", "folder.updated", "folder.deleted":
		return nuageFolderMessage(payload, icon, actor)
	case "share.created", "share.revoked":
		return nuageShareMessage(payload, icon, actor)
	}

	return fmt.Sprintf("ℹ️ <b>%s</b> triggered <b>%s</b>", actor, escape(payload.EventType))
}

// nuageSharedWith is the trailing " with someone@example.com" clause. It is
// empty when the payload carries no share, which is what keeps a bare event
// from rendering a dangling preposition.
func nuageSharedWith(payload nuagePayload, preposition string) string {
	share := payload.Data.Share
	if share == nil || share.SharedWithEmail == "" {
		return ""
	}
	return " " + preposition + " " + escape(share.SharedWithEmail)
}

// nuageFileMessage returns "" when the event names a file the payload does not
// carry: an alert naming nothing is worse than no alert.
func nuageFileMessage(payload nuagePayload, icon, actor string) string {
	file := payload.Data.File
	if file == nil {
		return ""
	}
	name := escape(file.Name)

	switch payload.EventType {
	case "file.uploaded":
		return fmt.Sprintf("%s <b>%s</b> uploaded <b>%s</b> (%s)", icon, actor, name, formatBytes(file.Size))
	case "file.updated":
		return fmt.Sprintf("%s <b>%s</b> updated <b>%s</b>", icon, actor, name)
	case "file.deleted":
		return fmt.Sprintf("%s <b>%s</b> deleted <b>%s</b>", icon, actor, name)
	}
	return fmt.Sprintf("%s <b>%s</b> shared <b>%s</b>%s", icon, actor, name, nuageSharedWith(payload, "with"))
}

func nuageFolderMessage(payload nuagePayload, icon, actor string) string {
	folder := payload.Data.Folder
	if folder == nil {
		return ""
	}
	name := escape(folder.Name)

	switch payload.EventType {
	case "folder.created":
		return fmt.Sprintf("%s <b>%s</b> created folder <b>%s</b>", icon, actor, name)
	case "folder.updated":
		return fmt.Sprintf("%s <b>%s</b> updated folder <b>%s</b>", icon, actor, name)
	}
	return fmt.Sprintf("%s <b>%s</b> deleted folder <b>%s</b>", icon, actor, name)
}

func nuageShareMessage(payload nuagePayload, icon, actor string) string {
	if payload.EventType == "share.revoked" {
		return fmt.Sprintf("%s <b>%s</b> revoked share access%s", icon, actor, nuageSharedWith(payload, "for"))
	}

	name := "a resource"
	if file := payload.Data.File; file != nil {
		name = escape(file.Name)
	}
	return fmt.Sprintf("%s <b>%s</b> shared <b>%s</b>%s", icon, actor, name, nuageSharedWith(payload, "with"))
}

func nuageEmoji(eventType string) string {
	switch eventType {
	case "file.uploaded":
		return "📁"
	case "file.updated", "folder.updated":
		return "✏️"
	case "file.deleted", "folder.deleted":
		return "🗑️"
	case "file.shared", "share.created":
		return "🔗"
	case "folder.created":
		return "📂"
	case "share.revoked":
		return "🚫"
	}
	return "ℹ️"
}

func formatBytes(size int64) string {
	const unit = 1024
	switch {
	case size < unit:
		return fmt.Sprintf("%d B", size)
	case size < unit*unit:
		return fmt.Sprintf("%.1f KB", float64(size)/unit)
	case size < unit*unit*unit:
		return fmt.Sprintf("%.1f MB", float64(size)/(unit*unit))
	}
	return fmt.Sprintf("%.1f GB", float64(size)/(unit*unit*unit))
}
