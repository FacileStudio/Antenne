package webhooks

import (
	"testing"

	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// These pin the rendered message of every branch the big provider formatters
// carry. They exist so that reshaping those formatters is provably a no-op:
// the message is what lands in a Matrix room, and a silent wording change there
// is invisible in review and obvious to whoever is on call.
type formatterCase struct {
	name     string
	provider settings.ProviderType
	payload  string
	want     string
}

var formatterCases = []formatterCase{
	{"vikunja/task.created", settings.ProviderVikunja,
		`{"event_name":"task.created","time":"2026-01-01T00:00:00Z","data":{"task":{"id":7,"title":"Ship it","identifier":"#7"},"doer":{"name":"Ada"}}}`, "📋 <b>[Vikunja]</b> Ada created task <code>#7</code> <b>Ship it</b>"},
	{"vikunja/task.updated done", settings.ProviderVikunja,
		`{"event_name":"task.updated","data":{"task":{"id":7,"title":"Ship it","done":true},"doer":{"username":"ada"}}}`, "📋 <b>[Vikunja]</b> ada completed task <code>#7</code> <b>Ship it</b>"},
	{"vikunja/task.updated reopened", settings.ProviderVikunja,
		`{"event_name":"task.updated","data":{"task":{"id":7,"title":"Ship it","done":false},"doer":{}}}`, "📋 <b>[Vikunja]</b> someone reopened task <code>#7</code> <b>Ship it</b>"},
	{"vikunja/task.updated unknown", settings.ProviderVikunja,
		`{"event_name":"task.updated","data":{"task":{"id":7,"title":"Ship it"}}}`, "📋 <b>[Vikunja]</b> someone updated task <code>#7</code> <b>Ship it</b>"},
	{"vikunja/assignee.created", settings.ProviderVikunja,
		`{"event_name":"task.assignee.created","data":{"task":{"id":7,"title":"Ship it","identifier":"T-7"},"doer":{"name":"Ada"},"assignee":{"name":"Bob"}}}`, "👤 <b>[Vikunja]</b> Ada assigned Bob to <code>T-7</code> <b>Ship it</b>"},
	{"vikunja/comment.created", settings.ProviderVikunja,
		`{"event_name":"task.comment.created","data":{"task":{"id":7},"doer":{"name":"Ada"}}}`, "💬 <b>[Vikunja]</b> Ada commented on <code>#7</code> <b>untitled</b>"},
	{"vikunja/attachment.deleted", settings.ProviderVikunja,
		`{"event_name":"task.attachment.deleted","data":{"task":{"title":"A & B"},"doer":{"name":"Ada"}}}`, "📎 <b>[Vikunja]</b> Ada removed an attachment from <b>A &amp; B</b>"},
	{"vikunja/project.updated", settings.ProviderVikunja,
		`{"event_name":"project.updated","data":{"project":{"id":3,"title":"Roadmap"},"doer":{"name":"Ada"}}}`, "📁 <b>[Vikunja]</b> Ada updated project <b>Roadmap</b>"},
	{"vikunja/project.shared.team", settings.ProviderVikunja,
		`{"event_name":"project.shared.team","data":{"project":{"id":3,"title":"Roadmap"},"doer":{"name":"Ada"}}}`, "🔗 <b>[Vikunja]</b> Ada shared project <b>Roadmap</b> with a team"},
	{"vikunja/team.created", settings.ProviderVikunja,
		`{"event_name":"team.created","data":{"team":{"id":2,"name":"Ops"},"doer":{"name":"Ada"}}}`, "👥 <b>[Vikunja]</b> Ada created team <b>Ops</b>"},
	{"vikunja/team.member.removed", settings.ProviderVikunja,
		`{"event_name":"team.member.removed","data":{"team":{"id":2,"name":"Ops"},"doer":{"name":"Ada"},"member":{"name":"Bob"}}}`, "👥 <b>[Vikunja]</b> Ada removed Bob in team <b>Ops</b>"},
	{"vikunja/unknown", settings.ProviderVikunja,
		`{"event_name":"label.created","data":{"doer":{"name":"Ada"}}}`, "📋 <b>[Vikunja]</b> Ada triggered <code>label.created</code>"},

	{"opus/task.created", settings.ProviderOpus,
		`{"event":"task.created","task":{"number":12,"title":"Ship it"},"project":{"name":"Suite"},"actor":{"name":"Ada"}}`, "📋 <b>[Opus]</b> Ada created task <code>#12</code> <b>Ship it</b> in <b>Suite</b>"},
	{"opus/status_changed", settings.ProviderOpus,
		`{"event":"task.status_changed","task":{"number":12,"title":"Ship it"},"actor":{"name":"Ada"},"data":{"oldStatus":"todo","newStatus":"doing"}}`, "🔄 <b>[Opus]</b> Ada changed status of <code>#12</code> <b>Ship it</b> from <code>todo</code> to <code>doing</code>"},
	{"opus/priority_changed no data", settings.ProviderOpus,
		`{"event":"task.priority_changed","task":{"title":"Ship it"},"actor":{"name":"Ada"}}`, "🏷️ <b>[Opus]</b> Ada changed priority of <b>Ship it</b>"},
	{"opus/title_changed both", settings.ProviderOpus,
		`{"event":"task.title_changed","task":{"number":1,"title":"New"},"actor":{"name":"Ada"},"data":{"oldTitle":"Old","newTitle":"New"}}`, "✏️ <b>[Opus]</b> Ada renamed task <code>#1</code> <b>Old</b> → <b>New</b>"},
	{"opus/title_changed partial", settings.ProviderOpus,
		`{"event":"task.title_changed","task":{"title":"New"},"actor":{"name":"Ada"},"data":{"oldTitle":"Old"}}`, "✏️ <b>[Opus]</b> Ada renamed task <b>New</b>"},
	{"opus/assignee_changed", settings.ProviderOpus,
		`{"event":"task.assignee_changed","task":{"title":"Ship it"},"actor":{"name":"Ada"},"data":{"newAssignee":"Bob"}}`, "👤 <b>[Opus]</b> Ada assigned <b>Ship it</b> to <b>Bob</b>"},
	{"opus/unassigned", settings.ProviderOpus,
		`{"event":"task.unassigned","task":{"title":"Ship it"},"actor":{"name":"Ada"},"data":{"oldAssignee":"Bob"}}`, "👤 <b>[Opus]</b> Ada unassigned <b>Ship it</b> from <b>Bob</b>"},
	{"opus/description_changed", settings.ProviderOpus,
		`{"event":"task.description_changed","task":{"title":"Ship it"},"project":{"name":"Suite"}}`, "📝 <b>[Opus]</b> someone updated description of <b>Ship it</b> in <b>Suite</b>"},
	{"opus/comment_created", settings.ProviderOpus,
		`{"event":"task.comment_created","task":{"title":"Ship it"},"actor":{"name":"Ada"}}`, "💬 <b>[Opus]</b> Ada commented on <b>Ship it</b>"},
	{"opus/unknown", settings.ProviderOpus,
		`{"event":"task.archived","task":{"title":"Ship it"},"actor":{"name":"Ada"}}`, "📋 <b>[Opus]</b> Ada triggered <code>task.archived</code> on <b>Ship it</b>"},

	{"nuage/file.uploaded", settings.ProviderNuage,
		`{"event_type":"file.uploaded","actor":{"name":"Ada"},"data":{"file":{"id":1,"name":"plan.pdf","size":2048}}}`, "📁 <b>Ada</b> uploaded <b>plan.pdf</b> (2.0 KB)"},
	{"nuage/file.uploaded missing file", settings.ProviderNuage,
		`{"event_type":"file.uploaded","actor":{"name":"Ada"},"data":{}}`, ""},
	{"nuage/file.updated", settings.ProviderNuage,
		`{"event_type":"file.updated","actor":{"name":"Ada"},"data":{"file":{"id":1,"name":"plan.pdf"}}}`, "✏️ <b>Ada</b> updated <b>plan.pdf</b>"},
	{"nuage/file.deleted", settings.ProviderNuage,
		`{"event_type":"file.deleted","actor":{"name":"Ada"},"data":{"file":{"id":1,"name":"plan.pdf"}}}`, "🗑️ <b>Ada</b> deleted <b>plan.pdf</b>"},
	{"nuage/file.shared", settings.ProviderNuage,
		`{"event_type":"file.shared","actor":{"name":"Ada"},"data":{"file":{"id":1,"name":"plan.pdf"},"share":{"shared_with_email":"bob@example.com"}}}`, "🔗 <b>Ada</b> shared <b>plan.pdf</b> with bob@example.com"},
	{"nuage/folder.created", settings.ProviderNuage,
		`{"event_type":"folder.created","actor":{"name":"Ada"},"data":{"folder":{"id":2,"name":"Docs"}}}`, "📂 <b>Ada</b> created folder <b>Docs</b>"},
	{"nuage/folder.updated", settings.ProviderNuage,
		`{"event_type":"folder.updated","actor":{"name":"Ada"},"data":{"folder":{"id":2,"name":"Docs"}}}`, "✏️ <b>Ada</b> updated folder <b>Docs</b>"},
	{"nuage/folder.deleted missing", settings.ProviderNuage,
		`{"event_type":"folder.deleted","actor":{"name":"Ada"},"data":{"file":{"id":1,"name":"x"}}}`, ""},
	{"nuage/share.created no file", settings.ProviderNuage,
		`{"event_type":"share.created","actor":{"name":"Ada"},"data":{"share":{"shared_with_email":"bob@example.com"}}}`, "🔗 <b>Ada</b> shared <b>a resource</b> with bob@example.com"},
	{"nuage/share.revoked", settings.ProviderNuage,
		`{"event_type":"share.revoked","actor":{"name":"Ada"},"data":{"share":{"shared_with_email":"bob@example.com"}}}`, "🚫 <b>Ada</b> revoked share access for bob@example.com"},
	{"nuage/unknown", settings.ProviderNuage,
		`{"event_type":"quota.exceeded","actor":{"name":"Ada"},"data":{}}`, "ℹ️ <b>Ada</b> triggered <b>quota.exceeded</b>"},

	{"ardoise/admin.client.created", settings.ProviderArdoise,
		`{"event":"admin.client.created","actor":{"email":"ada@example.com"},"data":{"entityId":"c1","record":{"id":"c1","email":"bob@example.com","firstName":"Bob","lastName":"Stone"}}}`, "<b>[Ardoise]</b> ada@example.com created client<br/>ID: <code>c1</code><br/>Name: <b>Bob Stone</b><br/>Email: <code>bob@example.com</code>"},
	{"ardoise/admin no data", settings.ProviderArdoise,
		`{"event":"admin.invoice.deleted","actor":{"id":"u1"}}`, "<b>[Ardoise]</b> u1 deleted invoice"},
	{"ardoise/settings.logo.updated", settings.ProviderArdoise,
		`{"event":"settings.logo.updated","actor":{"email":"ada@example.com"}}`, "🖼️ <b>[Ardoise]</b> logo updated"},
	{"ardoise/settings.updated", settings.ProviderArdoise,
		`{"event":"settings.updated","actor":{"email":"ada@example.com"},"data":{"changedFields":["vat","iban"]}}`, "⚙️ <b>[Ardoise]</b> settings updated<br/>Fields: <code>vat</code>, <code>iban</code>"},
	{"ardoise/other", settings.ProviderArdoise,
		`{"event":"invoice.sent","actor":{"email":"ada@example.com"}}`, "📡 <b>[Ardoise]</b> Invoice Sent"},

	{"docuseal/form.completed", settings.ProviderDocuSeal,
		`{"event_type":"form.completed","data":{"id":5,"name":"Bob","email":"bob@example.com","external_id":"ext-1","template":{"id":9,"name":"NDA"},"submission":{"id":11,"url":"https://ds.example/s/11"},"documents":[{"name":"NDA.pdf","url":"https://ds.example/d.pdf"},{"url":""}]}}`, "<b>[DocuSeal] NDA</b> completed<br/>Signer: <b>Bob</b><br/>Submission: <code>11</code><br/>External ID: <code>ext-1</code><br/><a href=\"https://ds.example/s/11\">Open submission</a><br/>Documents:<br/>• <a href=\"https://ds.example/d.pdf\">NDA.pdf</a>"},
	{"docuseal/form.declined", settings.ProviderDocuSeal,
		`{"event_type":"form.declined","data":{"id":5,"name":"Bob","decline_reason":"wrong terms","template":{"name":"NDA"},"submission":{"id":11}}}`, "<b>[DocuSeal] NDA</b> declined<br/>Signer: <b>Bob</b><br/>Reason: <i>wrong terms</i><br/>Submission: <code>11</code>"},
	{"docuseal/form.started", settings.ProviderDocuSeal,
		`{"event_type":"form.started","data":{"id":5,"name":"Bob","template":{"name":"NDA"},"submission":{"id":11}}}`, "<b>[DocuSeal] NDA</b> started<br/>Signer: <b>Bob</b><br/>Submission: <code>11</code>"},
	{"docuseal/form.viewed", settings.ProviderDocuSeal,
		`{"event_type":"form.viewed","data":{"id":5,"email":"bob@example.com","template":{"name":"NDA"}}}`, "<b>[DocuSeal] NDA</b> viewed<br/>Signer: <b>bob@example.com</b><br/>Submission: <code>5</code>"},
	{"docuseal/template.created", settings.ProviderDocuSeal,
		`{"event_type":"template.created","data":{"id":9,"name":"NDA","slug":"nda"}}`, "🧾 <b>[DocuSeal] Template created</b><br/>Name: <b>NDA</b><br/>Template ID: <code>9</code><br/>Slug: <code>nda</code>"},
	{"docuseal/submission.created", settings.ProviderDocuSeal,
		`{"event_type":"submission.created","data":{"id":11,"template":{"name":"NDA"},"submission":{"id":11}}}`, "📨 <b>[DocuSeal] Submission created</b><br/>Template: <b>NDA</b><br/>Submission: <code>11</code>"},
	{"docuseal/unknown", settings.ProviderDocuSeal,
		`{"event_type":"form.archived","data":{"id":1}}`, "ℹ️ <b>[DocuSeal]</b> form.archived"},

	{"git/push github", settings.ProviderGit,
		`{"ref":"refs/heads/main","before":"a1","after":"b2","compare":"https://gh/compare","commits":[{"id":"b2","message":"fix: thing\nbody","url":"https://gh/c","author":{"name":"Ada"}}],"head_commit":{"id":"b2","message":"fix: thing\nbody"},"pusher":{"name":"Ada"},"repository":{"full_name":"org/repo","html_url":"https://gh/repo"}}`, "<b>[org/repo]</b> Ada pushed 1 commit to <code>main</code><br/>  <code>b2</code> fix: thing<br/><a href=\"https://gh/compare\">Compare</a>"},
	{"git/push gitlab", settings.ProviderGit,
		`{"ref":"refs/heads/dev","after":"c3","total_commits_count":3,"user_name":"Ada","project":{"path_with_namespace":"org/repo","web_url":"https://gl/repo"},"commits":[{"id":"c3","message":"chore"}]}`, "<b>[org/repo]</b> Ada pushed 1 commit to <code>dev</code><br/>  <code>c3</code> chore"},
	{"git/pull_request merged", settings.ProviderGit,
		`{"action":"closed","pull_request":{"title":"Add thing","html_url":"https://gh/pr/4","number":4,"merged":true},"repository":{"full_name":"org/repo"},"sender":{"login":"ada"}}`, "<b>[org/repo]</b> ada closed PR <a href=\"https://gh/pr/4\">#4</a>: Add thing"},
	{"git/pull_request opened", settings.ProviderGit,
		`{"action":"opened","pull_request":{"title":"Add thing","html_url":"https://gh/pr/4","number":4},"repository":{"full_name":"org/repo"},"sender":{"login":"ada"}}`, "<b>[org/repo]</b> ada opened PR <a href=\"https://gh/pr/4\">#4</a>: Add thing"},
	{"git/issue opened", settings.ProviderGit,
		`{"action":"opened","issue":{"title":"Broken","html_url":"https://gh/i/2","number":2},"repository":{"full_name":"org/repo"},"sender":{"username":"ada"}}`, "<b>[org/repo]</b> ada opened issue <a href=\"https://gh/i/2\">#2</a>: Broken"},
	{"git/unknown", settings.ProviderGit,
		`{"repository":{"name":"repo"},"user":{"name":"Ada"}}`, "📡 <b>[repo]</b> Ada triggered a webhook event"},
}

func TestProviderMessagesAreUnchanged(t *testing.T) {
	for _, testCase := range formatterCases {
		definition, known := Lookup(testCase.provider)
		if !known {
			t.Fatalf("%s: no definition for %s", testCase.name, testCase.provider)
		}
		result, err := definition.Handle([]byte(testCase.payload), Context{})
		if err != nil {
			t.Fatalf("%s: handle: %v", testCase.name, err)
		}
		if result.MessageHTML != testCase.want {
			t.Errorf("%s\n got: %s\nwant: %s", testCase.name, result.MessageHTML, testCase.want)
		}
	}
}
