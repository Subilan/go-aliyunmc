package forum_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/perm_util"
	"github.com/Subilan/go-aliyunmc/internal/common/session_util"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/common/testutil"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum"
	forumsession "github.com/Subilan/go-aliyunmc/internal/modules/forum/session"

	"github.com/creasty/defaults"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type testClient struct {
	t    *testing.T
	base string
	http *http.Client
}

func newClient(t *testing.T, base string) *testClient {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	return &testClient{t: t, base: base, http: &http.Client{Jar: jar}}
}

func (c *testClient) do(method, path string, body any) (int, map[string]any) {
	c.t.Helper()

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	payload := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			c.t.Fatalf("%s %s: invalid JSON %q: %v", method, path, string(raw), err)
		}
	}
	return resp.StatusCode, payload
}

// doRaw 发送原始请求，用于验证 Content-Type、Origin 与请求体上限。
func (c *testClient) doRaw(method, path, contentType, origin string, body []byte) (int, map[string]any) {
	c.t.Helper()

	req, err := http.NewRequest(method, c.base+path, bytes.NewReader(body))
	if err != nil {
		c.t.Fatalf("new raw request: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	payload := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &payload)
	}
	return resp.StatusCode, payload
}

func (c *testClient) mustDo(method, path string, body any, wantStatus int) map[string]any {
	c.t.Helper()
	status, payload := c.do(method, path, body)
	if status != wantStatus {
		c.t.Fatalf("%s %s: status = %d, want %d, body = %v", method, path, status, wantStatus, payload)
	}
	return payload
}

func (c *testClient) upload(kind string, file []byte) map[string]any {
	c.t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.WriteField("kind", kind); err != nil {
		c.t.Fatalf("write field: %v", err)
	}
	part, err := writer.CreateFormFile("file", "test.png")
	if err != nil {
		c.t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(file); err != nil {
		c.t.Fatalf("write file: %v", err)
	}
	_ = writer.Close()

	req, err := http.NewRequest(http.MethodPost, c.base+"/forum-v1/uploads", &buf)
	if err != nil {
		c.t.Fatalf("new upload request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("upload: %v", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	payload := map[string]any{}
	_ = json.Unmarshal(raw, &payload)
	if resp.StatusCode != http.StatusOK {
		c.t.Fatalf("upload: status = %d, body = %v", resp.StatusCode, payload)
	}
	return dataMap(c.t, payload)
}

func dataMap(t *testing.T, payload map[string]any) map[string]any {
	t.Helper()
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("response has no data object: %v", payload)
	}
	return data
}

func items(t *testing.T, payload map[string]any) []any {
	t.Helper()
	data := dataMap(t, payload)
	list, ok := data["items"].([]any)
	if !ok {
		t.Fatalf("response has no items array: %v", payload)
	}
	return list
}

func num(t *testing.T, m map[string]any, key string) uint {
	t.Helper()
	value, ok := m[key].(float64)
	if !ok {
		t.Fatalf("field %q is not numeric: %v", key, m)
	}
	return uint(value)
}

func str(t *testing.T, m map[string]any, key string) string {
	t.Helper()
	value, ok := m[key].(string)
	if !ok {
		t.Fatalf("field %q is not string: %v", key, m)
	}
	return value
}

func TestForumE2E(t *testing.T) {
	cleanup, err := testutil.OpenMySQLTestDB()
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(cleanup)

	cfg := &config.Config{}
	if err := defaults.Set(cfg); err != nil {
		t.Fatalf("defaults: %v", err)
	}
	cfg.Forum.UploadDir = t.TempDir()
	cfg.Forum.RateLimit.Enabled = false
	config.Set(cfg)

	if err := session_util.InitStore(store.DB, forumsession.Manager, forumsession.AdminManager); err != nil {
		t.Fatalf("init session store: %v", err)
	}

	forum.MustInitialize()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	forum.Bind(engine)
	handler := forumsession.Manager.LoadAndSave(forumsession.AdminManager.LoadAndSave(engine))

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	superuser := createUser(t, "admin", "password", perm_util.RoleSuperuser)
	member := createUser(t, "member", "password", perm_util.RoleBasic)
	moderator := createUser(t, "moderator", "password", perm_util.RoleBasic)
	pending := createUser(t, "pending", "password", perm_util.RoleSuperuser)

	adminClient := newClient(t, server.URL)
	adminClient.mustDo(http.MethodPost, "/forum-v1/admin/auth/login", map[string]any{
		"username": "admin", "password": "password",
	}, http.StatusOK)

	boardPayload := adminClient.mustDo(http.MethodPost, "/forum-v1/admin/boards", map[string]any{
		"slug": "general", "name": "综合讨论", "description": "随便聊聊", "icon": "message-text", "color": "#6366f1",
	}, http.StatusOK)
	boardID := num(t, dataMap(t, boardPayload), "id")

	secondPayload := adminClient.mustDo(http.MethodPost, "/forum-v1/admin/boards", map[string]any{
		"slug": "life", "name": "生活",
	}, http.StatusOK)
	secondBoardID := num(t, dataMap(t, secondPayload), "id")

	memberClient := newClient(t, server.URL)
	memberClient.mustDo(http.MethodPost, "/forum-v1/auth/login", map[string]any{
		"username": "member", "password": "password", "remember": true,
	}, http.StatusOK)
	superClient := newClient(t, server.URL)
	superClient.mustDo(http.MethodPost, "/forum-v1/auth/login", map[string]any{
		"username": "admin", "password": "password",
	}, http.StatusOK)

	// 从未登录过论坛前台的共享账号在论坛与后台用户管理中都视为不存在。
	// 后台登录会初始化资料与默认权限，但不产生论坛登录记录，因此仍应隐藏。
	pendingClient := newClient(t, server.URL)
	pendingClient.mustDo(http.MethodPost, "/forum-v1/admin/auth/login", map[string]any{
		"username": "pending", "password": "password",
	}, http.StatusOK)
	memberClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/users/%d", pending.ID), nil, http.StatusNotFound)
	memberClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/users/%d/threads", pending.ID), nil, http.StatusNotFound)
	memberClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/users/%d/replies", pending.ID), nil, http.StatusNotFound)
	pendingUsers := adminClient.mustDo(http.MethodGet, "/forum-v1/admin/users?keyword=pending", nil, http.StatusOK)
	if len(items(t, pendingUsers)) != 0 {
		t.Fatalf("pending users = %v", pendingUsers)
	}
	adminClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/admin/users/%d", pending.ID), nil, http.StatusNotFound)
	adminClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/admin/users/%d/login-records", pending.ID), nil, http.StatusNotFound)
	for _, item := range items(t, memberClient.mustDo(http.MethodGet, "/forum-v1/users/search?q=pending", nil, http.StatusOK)) {
		if num(t, item.(map[string]any), "id") == pending.ID {
			t.Fatalf("pending user in mention search: %v", item)
		}
	}

	imageURL := uploadPNG(t, memberClient)
	threadContent := fmt.Sprintf(`<p>你好世界 #初见 <span class="mention" data-tag-type="mention" data-user-id="%d">@admin</span></p><p><img src="%s" alt="图"></p>`, superuser.ID, imageURL)
	threadPayload := memberClient.mustDo(http.MethodPost, "/forum-v1/threads", map[string]any{
		"boardId": boardID, "title": "第一篇主题", "contentHtml": threadContent,
	}, http.StatusOK)
	threadDetail := dataMap(t, threadPayload)
	threadID := num(t, threadDetail, "id")

	if got := str(t, threadDetail, "contentHtml"); !strings.Contains(got, "@admin") {
		t.Fatalf("mention chip not normalized: %s", got)
	}
	tags, _ := threadDetail["tags"].([]any)
	if len(tags) != 1 || tags[0] != "初见" {
		t.Fatalf("tags = %v", tags)
	}
	if board := threadDetail["board"].(map[string]any); board["slug"] != "general" {
		t.Fatalf("board = %v", board)
	}

	listPayload := memberClient.mustDo(http.MethodGet, "/forum-v1/threads?board=general", nil, http.StatusOK)
	if len(items(t, listPayload)) != 1 {
		t.Fatalf("thread list = %v", listPayload)
	}
	if listPayload["data"].(map[string]any)["total"].(float64) != 1 {
		t.Fatalf("total = %v", listPayload)
	}
	cappedList := memberClient.mustDo(http.MethodGet, "/forum-v1/threads?board=general&pageSize=100", nil, http.StatusOK)
	if cappedList["data"].(map[string]any)["pageSize"].(float64) != 50 {
		t.Fatalf("public page size not capped at 50: %v", cappedList)
	}

	// 提及通知：主题作者 basic，被提及者 superuser。
	notifications := superClient.mustDo(http.MethodGet, "/forum-v1/notifications", nil, http.StatusOK)
	if len(items(t, notifications)) != 1 {
		t.Fatalf("superuser notifications = %v", notifications)
	}
	mention := items(t, notifications)[0].(map[string]any)
	if mention["type"] != "mention" || num(t, mention, "threadId") != threadID {
		t.Fatalf("mention notification = %v", mention)
	}
	unread := superClient.mustDo(http.MethodGet, "/forum-v1/notifications/unread-count", nil, http.StatusOK)
	if unread["data"].(map[string]any)["count"].(float64) != 1 {
		t.Fatalf("unread count = %v", unread)
	}

	reply1Payload := superClient.mustDo(http.MethodPost, fmt.Sprintf("/forum-v1/threads/%d/replies", threadID), map[string]any{
		"contentHtml": `<p>第一条回复</p>`,
	}, http.StatusOK)
	reply1 := dataMap(t, reply1Payload)
	reply1ID := num(t, reply1, "id")
	if num(t, reply1, "floor") != 1 {
		t.Fatalf("floor = %v", reply1Payload)
	}

	// 回复主题作者（basic）应收到 reply 通知。
	memberNotifications := memberClient.mustDo(http.MethodGet, "/forum-v1/notifications", nil, http.StatusOK)
	foundReply := false
	for _, raw := range items(t, memberNotifications) {
		item := raw.(map[string]any)
		if item["type"] == "reply" && num(t, item, "replyId") == reply1ID {
			foundReply = true
		}
	}
	if !foundReply {
		t.Fatalf("reply notification missing: %v", memberNotifications)
	}

	reply2Payload := memberClient.mustDo(http.MethodPost, fmt.Sprintf("/forum-v1/threads/%d/replies", threadID), map[string]any{
		"contentHtml": fmt.Sprintf(`<p><span class="reply-reference" data-tag-type="reply" data-floor="1">#1</span> 引用回复</p>`),
	}, http.StatusOK)
	reply2 := dataMap(t, reply2Payload)
	reply2ID := num(t, reply2, "id")
	if num(t, reply2, "floor") != 2 {
		t.Fatalf("floor = %v", reply2)
	}
	if num(t, reply2, "replyToFloor") != 1 || num(t, reply2, "replyToReplyId") != reply1ID {
		t.Fatalf("reply reference = %v", reply2)
	}

	// 点赞：主题 -> basic，回复 -> superuser。
	likePayload := superClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/threads/%d/like", threadID), nil, http.StatusOK)
	if likePayload["data"].(map[string]any)["liked"] != true {
		t.Fatalf("like = %v", likePayload)
	}
	memberClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/replies/%d/like", reply1ID), nil, http.StatusOK)

	// 取消点赞保留通知（点赞关系被删除，通知不做撤回）。
	superClient.mustDo(http.MethodDelete, fmt.Sprintf("/forum-v1/replies/%d/like", reply1ID), nil, http.StatusOK)

	// 搜索与标签。
	searchPayload := memberClient.mustDo(http.MethodGet, "/forum-v1/search?q="+"你好", nil, http.StatusOK)
	if len(items(t, searchPayload)) != 1 {
		t.Fatalf("search = %v", searchPayload)
	}
	tagSearch := memberClient.mustDo(http.MethodGet, "/forum-v1/search?q=%23初见", nil, http.StatusOK)
	if len(items(t, tagSearch)) != 1 {
		t.Fatalf("tag search = %v", tagSearch)
	}
	bareHash := memberClient.mustDo(http.MethodGet, "/forum-v1/search?q=%23", nil, http.StatusOK)
	if len(items(t, bareHash)) != 0 {
		t.Fatalf("bare # should not return all threads: %v", bareHash)
	}
	tagsPayload := memberClient.mustDo(http.MethodGet, "/forum-v1/tags", nil, http.StatusOK)
	if len(items(t, tagsPayload)) == 0 {
		t.Fatalf("hot tags = %v", tagsPayload)
	}

	// 删除回复保留楼层占位，恢复后正文回来。
	superClient.mustDo(http.MethodDelete, fmt.Sprintf("/forum-v1/replies/%d", reply1ID), nil, http.StatusOK)
	repliesPayload := memberClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/threads/%d/replies", threadID), nil, http.StatusOK)
	replyItems := items(t, repliesPayload)
	if len(replyItems) != 2 {
		t.Fatalf("replies = %v", replyItems)
	}
	first := replyItems[0].(map[string]any)
	if first["deleted"] != true || first["contentHtml"] != nil || num(t, first, "floor") != 1 {
		t.Fatalf("deleted placeholder = %v", first)
	}
	superClient.mustDo(http.MethodPost, fmt.Sprintf("/forum-v1/replies/%d/restore", reply1ID), nil, http.StatusOK)

	// 置顶与后台列表。
	adminClient.mustDo(http.MethodPatch, fmt.Sprintf("/forum-v1/admin/threads/%d", threadID), map[string]any{"pinned": true}, http.StatusOK)
	pinned := memberClient.mustDo(http.MethodGet, "/forum-v1/pinned-threads?board=general", nil, http.StatusOK)
	if len(items(t, pinned)) != 1 {
		t.Fatalf("pinned = %v", pinned)
	}

	// 第二板块主题，用于检验板块级管理员过滤。
	memberClient.mustDo(http.MethodPost, "/forum-v1/threads", map[string]any{
		"boardId": secondBoardID, "title": "生活主题", "contentHtml": "<p>生活</p>",
	}, http.StatusOK)

	// 板块级管理员：只授权第一个板块。
	adminClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/admin/users/%d/permissions", moderator.ID), map[string]any{
		"grants": []map[string]any{
			{"permissionCode": "admin.access", "boardId": boardID},
			{"permissionCode": "content.view_admin", "boardId": boardID},
			{"permissionCode": "thread.pin", "boardId": boardID},
		},
	}, http.StatusOK)

	modAdmin := newClient(t, server.URL)
	modAdmin.mustDo(http.MethodPost, "/forum-v1/admin/auth/login", map[string]any{
		"username": "moderator", "password": "password",
	}, http.StatusOK)

	modThreads := modAdmin.mustDo(http.MethodGet, "/forum-v1/admin/threads", nil, http.StatusOK)
	modItems := items(t, modThreads)
	if len(modItems) != 1 || num(t, modItems[0].(map[string]any), "boardId") != boardID {
		t.Fatalf("board-scoped admin threads = %v", modThreads)
	}

	modForum := newClient(t, server.URL)
	modForum.mustDo(http.MethodPost, "/forum-v1/auth/login", map[string]any{
		"username": "moderator", "password": "password",
	}, http.StatusOK)
	modForum.mustDo(http.MethodPost, fmt.Sprintf("/forum-v1/threads/%d/pin", threadID), map[string]any{"pinned": false}, http.StatusOK)
	modAdmin.mustDo(http.MethodPost, "/forum-v1/admin/boards", map[string]any{"slug": "denied", "name": "拒绝"}, http.StatusForbidden)

	// 后台回复列表、用户列表、设置与登录记录。
	adminReplies := adminClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/admin/replies?threadId=%d", threadID), nil, http.StatusOK)
	if len(items(t, adminReplies)) == 0 {
		t.Fatalf("admin replies = %v", adminReplies)
	}
	adminUsers := adminClient.mustDo(http.MethodGet, "/forum-v1/admin/users?keyword=member", nil, http.StatusOK)
	if len(items(t, adminUsers)) != 1 {
		t.Fatalf("admin users = %v", adminUsers)
	}
	// 多个 permission 参数取“任一命中”。
	multiPermissionUsers := adminClient.mustDo(http.MethodGet, "/forum-v1/admin/users?permission=thread.pin&permission=board.create", nil, http.StatusOK)
	foundModerator := false
	for _, item := range items(t, multiPermissionUsers) {
		if num(t, item.(map[string]any), "id") == moderator.ID {
			foundModerator = true
			break
		}
	}
	if !foundModerator {
		t.Fatalf("multi-permission user filter = %v", multiPermissionUsers)
	}
	adminClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/admin/users/%d", member.ID), nil, http.StatusOK)
	adminClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/admin/users/%d/login-records", member.ID), nil, http.StatusOK)
	adminClient.mustDo(http.MethodGet, "/forum-v1/admin/settings", nil, http.StatusOK)
	adminClient.mustDo(http.MethodGet, "/forum-v1/admin/permissions/thread.pin/users?boardId="+uintString(boardID), nil, http.StatusOK)
	userReplies := memberClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/users/%d/replies", member.ID), nil, http.StatusOK)
	userReplyItems := items(t, userReplies)
	if len(userReplyItems) == 0 {
		t.Fatalf("user replies = %v", userReplies)
	}
	if title := userReplyItems[0].(map[string]any)["threadTitle"]; title != "第一篇主题" {
		t.Fatalf("user reply thread title = %v", title)
	}

	// 封禁：写操作立即 403，解封后恢复。
	adminClient.mustDo(http.MethodPost, fmt.Sprintf("/forum-v1/admin/users/%d/ban", member.ID), nil, http.StatusOK)
	memberClient.mustDo(http.MethodPost, "/forum-v1/threads", map[string]any{
		"boardId": boardID, "title": "封禁后", "contentHtml": "<p>x</p>",
	}, http.StatusForbidden)
	adminClient.mustDo(http.MethodPost, fmt.Sprintf("/forum-v1/admin/users/%d/unban", member.ID), nil, http.StatusOK)

	// allow_guest_read：关掉后公开读接口 401，/site 仍可访问。
	adminClient.mustDo(http.MethodPut, "/forum-v1/admin/settings", map[string]any{
		"allow_guest_read": "false",
	}, http.StatusOK)
	guest := newClient(t, server.URL)
	guest.mustDo(http.MethodGet, "/forum-v1/threads", nil, http.StatusUnauthorized)
	guest.mustDo(http.MethodGet, "/forum-v1/site", nil, http.StatusOK)
	adminClient.mustDo(http.MethodPut, "/forum-v1/admin/settings", map[string]any{
		"allow_guest_read": "true",
	}, http.StatusOK)
	guest.mustDo(http.MethodGet, "/forum-v1/threads", nil, http.StatusOK)

	// 静态文件无需登录。
	fileResp, err := guest.http.Get(server.URL + imageURL)
	if err != nil {
		t.Fatalf("static file: %v", err)
	}
	_ = fileResp.Body.Close()
	if fileResp.StatusCode != http.StatusOK {
		t.Fatalf("static file status = %d", fileResp.StatusCode)
	}
	if contentType := fileResp.Header.Get("Content-Type"); contentType != "image/png" {
		t.Fatalf("static content-type = %q", contentType)
	}

	// 上传被正文引用时不能删除。
	uploadsPayload := adminClient.mustDo(http.MethodGet, "/forum-v1/admin/uploads", nil, http.StatusOK)
	uploadItems := items(t, uploadsPayload)
	if len(uploadItems) != 1 {
		t.Fatalf("admin uploads = %v", uploadsPayload)
	}
	uploadID := num(t, uploadItems[0].(map[string]any), "id")
	adminClient.mustDo(http.MethodDelete, fmt.Sprintf("/forum-v1/admin/uploads/%d", uploadID), nil, http.StatusConflict)

	statsPayload := adminClient.mustDo(http.MethodGet, "/forum-v1/admin/uploads/stats", nil, http.StatusOK)
	if len(items(t, statsPayload)) != 1 {
		t.Fatalf("upload stats = %v", statsPayload)
	}

	// 权限目录、看板与审计。
	permissionsPayload := adminClient.mustDo(http.MethodGet, "/forum-v1/admin/permissions", nil, http.StatusOK)
	if len(items(t, permissionsPayload)) != 34 {
		t.Fatalf("permission catalog = %v", permissionsPayload)
	}
	dashboardPayload := adminClient.mustDo(http.MethodGet, "/forum-v1/admin/dashboard?days=7", nil, http.StatusOK)
	dashboard := dataMap(t, dashboardPayload)
	trend, ok := dashboard["trend"].([]any)
	if !ok || len(trend) != 7 {
		t.Fatalf("dashboard trend = %v", dashboard["trend"])
	}
	todayTrend, ok := trend[len(trend)-1].(map[string]any)
	if !ok || num(t, todayTrend, "threads") < 1 || num(t, todayTrend, "replies") < 1 {
		t.Fatalf("dashboard today trend = %v", trend[len(trend)-1])
	}
	auditPayload := adminClient.mustDo(http.MethodGet, "/forum-v1/admin/audit-logs", nil, http.StatusOK)
	if len(items(t, auditPayload)) == 0 {
		t.Fatalf("audit logs empty: %v", auditPayload)
	}

	// 超级用户的 admin/me 应包含全部权限。
	mePayload := adminClient.mustDo(http.MethodGet, "/forum-v1/admin/auth/me", nil, http.StatusOK)
	me := dataMap(t, mePayload)
	if me["isSuperuser"] != true {
		t.Fatalf("admin me = %v", me)
	}

	// 普通用户访问后台 403/401。
	memberClient.mustDo(http.MethodGet, "/forum-v1/admin/auth/me", nil, http.StatusUnauthorized)

	// 用户搜索只返回有效用户。
	searchUsers := memberClient.mustDo(http.MethodGet, "/forum-v1/users/search?q=admin", nil, http.StatusOK)
	if len(items(t, searchUsers)) != 1 {
		t.Fatalf("mention search = %v", searchUsers)
	}

	// 安全边界：Content-Type、Origin、请求体上限与搜索上限。
	if status, _ := memberClient.doRaw(http.MethodPatch, "/forum-v1/me/profile", "text/plain", "", []byte(`{"bio":"x"}`)); status != http.StatusBadRequest {
		t.Fatalf("non-json content type status = %d", status)
	}
	if status, _ := memberClient.doRaw(http.MethodPatch, "/forum-v1/me/profile", "application/json", "https://evil.example", []byte(`{"bio":"x"}`)); status != http.StatusForbidden {
		t.Fatalf("cross-origin status = %d", status)
	}
	if status, _ := memberClient.doRaw(http.MethodPatch, "/forum-v1/me/profile", "application/json", server.URL, []byte(`{"bio":"x"}`)); status != http.StatusOK {
		t.Fatalf("same-origin status = %d", status)
	}
	bigBody := []byte(`{"boardId":` + uintString(boardID) + `,"title":"big","contentHtml":"` + strings.Repeat("a", 2<<20) + `"}`)
	if status, _ := memberClient.doRaw(http.MethodPost, "/forum-v1/threads", "application/json", "", bigBody); status != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body status = %d", status)
	}
	memberClient.mustDo(http.MethodGet, "/forum-v1/search?q=a+b+c+d+e+f", nil, http.StatusBadRequest)

	// 标签数量上限。
	manyTags := make([]string, 0, 21)
	for i := 0; i < 21; i++ {
		manyTags = append(manyTags, fmt.Sprintf("#tag%d", i))
	}
	memberClient.mustDo(http.MethodPost, "/forum-v1/threads", map[string]any{
		"boardId":     boardID,
		"title":       "标签过多",
		"contentHtml": "<p>" + strings.Join(manyTags, " ") + "</p>",
	}, http.StatusBadRequest)

	// 字段缺失不能被当作“显式置空”。
	memberClient.mustDo(http.MethodPut, "/forum-v1/me/avatar", map[string]any{}, http.StatusBadRequest)
	memberClient.mustDo(http.MethodPut, "/forum-v1/me/avatar", map[string]any{"uploadId": nil}, http.StatusOK)
	adminClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/admin/users/%d/permissions", moderator.ID), map[string]any{}, http.StatusBadRequest)

	// 通知批量标记有数量上限。
	tooManyIDs := make([]uint, 101)
	memberClient.mustDo(http.MethodPost, "/forum-v1/notifications/read", map[string]any{"ids": tooManyIDs}, http.StatusBadRequest)

	// 委派管理员不能封禁超级用户。
	adminClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/admin/users/%d/permissions", moderator.ID), map[string]any{
		"grants": []map[string]any{
			{"permissionCode": "admin.access", "boardId": boardID},
			{"permissionCode": "user.ban"},
		},
	}, http.StatusOK)
	modAdmin.mustDo(http.MethodPost, fmt.Sprintf("/forum-v1/admin/users/%d/ban", superuser.ID), nil, http.StatusForbidden)

	// 后台编辑/删除/恢复跟帖，以及重复删除的 409 语义。
	adminClient.mustDo(http.MethodPatch, fmt.Sprintf("/forum-v1/admin/replies/%d", reply2ID), map[string]any{
		"contentHtml": "<p>后台编辑</p>",
	}, http.StatusOK)

	replyDetail := dataMap(t, adminClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/admin/replies/%d", reply2ID), nil, http.StatusOK))
	if !strings.Contains(str(t, replyDetail, "contentHtml"), "后台编辑") {
		t.Fatalf("admin reply detail = %v", replyDetail)
	}
	// 没有 content.view_admin 的管理员不能读取跟帖详情。
	modAdmin.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/admin/replies/%d", reply2ID), nil, http.StatusForbidden)
	adminClient.mustDo(http.MethodGet, "/forum-v1/admin/replies/9999999", nil, http.StatusNotFound)

	adminClient.mustDo(http.MethodDelete, fmt.Sprintf("/forum-v1/admin/replies/%d", reply2ID), nil, http.StatusOK)
	deletedDetail := dataMap(t, adminClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/admin/replies/%d", reply2ID), nil, http.StatusOK))
	if deletedDetail["deleted"] != true || !strings.Contains(str(t, deletedDetail, "contentHtml"), "后台编辑") {
		t.Fatalf("deleted admin reply detail = %v", deletedDetail)
	}
	adminClient.mustDo(http.MethodDelete, fmt.Sprintf("/forum-v1/admin/replies/%d", reply2ID), nil, http.StatusConflict)
	adminClient.mustDo(http.MethodPost, fmt.Sprintf("/forum-v1/admin/replies/%d/restore", reply2ID), nil, http.StatusOK)
	adminClient.mustDo(http.MethodPost, "/forum-v1/admin/threads/batch", map[string]any{
		"ids": []uint{threadID}, "action": "pin",
	}, http.StatusOK)
	adminClient.mustDo(http.MethodPost, "/forum-v1/admin/threads/batch", map[string]any{
		"ids": []uint{threadID}, "action": "unpin",
	}, http.StatusOK)

	// 用户主页。
	profilePayload := memberClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/users/%d", member.ID), nil, http.StatusOK)
	profile := dataMap(t, profilePayload)
	if profile["isSelf"] != true || num(t, profile, "threadCount") != 2 {
		t.Fatalf("profile = %v", profile)
	}

	// 主页点赞数与内容获赞数分开统计。
	memberClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/users/%d/like", superuser.ID), nil, http.StatusOK)
	superProfile := dataMap(t, memberClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/users/%d", superuser.ID), nil, http.StatusOK))
	if num(t, superProfile, "profileLikeCount") != 1 || superProfile["likedByMe"] != true {
		t.Fatalf("profile like fields = %v", superProfile)
	}
	if num(t, superProfile, "likeCount") != 1 {
		t.Fatalf("content likeCount = %v", superProfile)
	}

	// 回归：删除主题下最后一条跟帖时 MAX(created_at) 为 NULL，
	// 后台必须能正常删除并把 lastReplyAt 清空，而不是 500。
	lastThread := dataMap(t, memberClient.mustDo(http.MethodPost, "/forum-v1/threads", map[string]any{
		"boardId":     boardID,
		"title":       "最后一条跟帖",
		"contentHtml": "<p>主题正文</p>",
	}, http.StatusOK))
	lastThreadID := uint(num(t, lastThread, "id"))
	lastReply := dataMap(t, memberClient.mustDo(http.MethodPost, fmt.Sprintf("/forum-v1/threads/%d/replies", lastThreadID), map[string]any{
		"contentHtml": "<p>唯一跟帖</p>",
	}, http.StatusOK))
	lastReplyID := uint(num(t, lastReply, "id"))
	memberClient.mustDo(http.MethodDelete, fmt.Sprintf("/forum-v1/replies/%d", lastReplyID), nil, http.StatusOK)

	lastThreadAdmin := dataMap(t, adminClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/admin/threads/%d", lastThreadID), nil, http.StatusOK))
	if lastThreadAdmin["lastReplyAt"] != nil {
		t.Fatalf("lastReplyAt = %v, want nil", lastThreadAdmin["lastReplyAt"])
	}

	// 首次登录论坛后，账号才在论坛个人页与后台用户管理中可见。
	pendingClient.mustDo(http.MethodPost, "/forum-v1/auth/login", map[string]any{
		"username": "pending", "password": "password",
	}, http.StatusOK)
	pendingClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/users/%d", pending.ID), nil, http.StatusOK)
	adminClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/admin/users/%d", pending.ID), nil, http.StatusOK)
}

func uploadPNG(t *testing.T, client *testClient) string {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for x := 0; x < 2; x++ {
		for y := 0; y < 2; y++ {
			img.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	result := client.upload("post", buf.Bytes())
	url := str(t, result, "url")
	if !strings.HasPrefix(url, "/forum-v1/files/") {
		t.Fatalf("upload url = %q", url)
	}
	return url
}

func uintString(value uint) string {
	return fmt.Sprintf("%d", value)
}

func createUser(t *testing.T, username, password string, role perm_util.Role) models.User {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}

	user := models.User{Username: username, PasswordHash: string(hash), Role: role}
	if err := store.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}
