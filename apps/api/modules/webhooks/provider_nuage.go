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
	file := payload.Data.File
	folder := payload.Data.Folder
	share := payload.Data.Share

	sharedWith := func(preposition string) string {
		if share == nil || share.SharedWithEmail == "" {
			return ""
		}
		return " " + preposition + " " + escape(share.SharedWithEmail)
	}

	switch payload.EventType {
	case "file.uploaded":
		if file == nil {
			return ""
		}
		return fmt.Sprintf("%s <b>%s</b> uploaded <b>%s</b> (%s)", icon, actor, escape(file.Name), formatBytes(file.Size))
	case "file.updated":
		if file == nil {
			return ""
		}
		return fmt.Sprintf("%s <b>%s</b> updated <b>%s</b>", icon, actor, escape(file.Name))
	case "file.deleted":
		if file == nil {
			return ""
		}
		return fmt.Sprintf("%s <b>%s</b> deleted <b>%s</b>", icon, actor, escape(file.Name))
	case "file.shared":
		if file == nil {
			return ""
		}
		return fmt.Sprintf("%s <b>%s</b> shared <b>%s</b>%s", icon, actor, escape(file.Name), sharedWith("with"))
	case "folder.created":
		if folder == nil {
			return ""
		}
		return fmt.Sprintf("%s <b>%s</b> created folder <b>%s</b>", icon, actor, escape(folder.Name))
	case "folder.updated":
		if folder == nil {
			return ""
		}
		return fmt.Sprintf("%s <b>%s</b> updated folder <b>%s</b>", icon, actor, escape(folder.Name))
	case "folder.deleted":
		if folder == nil {
			return ""
		}
		return fmt.Sprintf("%s <b>%s</b> deleted folder <b>%s</b>", icon, actor, escape(folder.Name))
	case "share.created":
		name := "a resource"
		if file != nil {
			name = escape(file.Name)
		}
		return fmt.Sprintf("%s <b>%s</b> shared <b>%s</b>%s", icon, actor, name, sharedWith("with"))
	case "share.revoked":
		return fmt.Sprintf("%s <b>%s</b> revoked share access%s", icon, actor, sharedWith("for"))
	}

	return fmt.Sprintf("ℹ️ <b>%s</b> triggered <b>%s</b>", actor, escape(payload.EventType))
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
