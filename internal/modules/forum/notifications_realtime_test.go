package forum_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/perm_util"
	"github.com/Subilan/go-aliyunmc/internal/common/session_util"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/common/testutil"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
	forumsession "github.com/Subilan/go-aliyunmc/internal/modules/forum/session"

	"github.com/creasty/defaults"
	"github.com/gin-gonic/gin"
)

type notificationEnv struct {
	t           *testing.T
	server      *httptest.Server
	admin       models.User
	alice       models.User
	bob         models.User
	carol       models.User
	dave        models.User
	adminClient *testClient
	aliceClient *testClient
	bobClient   *testClient
	carolClient *testClient
	daveClient  *testClient
	boardID     uint
	boardSlug   string
}

func setupNotificationEnv(t *testing.T) *notificationEnv {
	t.Helper()

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
	cfg.Forum.LikeNotificationCooldownMinutes = 10
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

	admin := createUser(t, "admin", "password", perm_util.RoleSuperuser)
	alice := createUser(t, "alice", "password", perm_util.RoleBasic)
	bob := createUser(t, "bob", "password", perm_util.RoleBasic)
	carol := createUser(t, "carol", "password", perm_util.RoleBasic)
	dave := createUser(t, "dave", "password", perm_util.RoleBasic)

	env := &notificationEnv{
		t:           t,
		server:      server,
		admin:       admin,
		alice:       alice,
		bob:         bob,
		carol:       carol,
		dave:        dave,
		adminClient: probeLogin(t, server.URL, "/forum-v1/admin/auth/login", "admin"),
		aliceClient: probeLogin(t, server.URL, "/forum-v1/auth/login", "alice"),
		bobClient:   probeLogin(t, server.URL, "/forum-v1/auth/login", "bob"),
		carolClient: probeLogin(t, server.URL, "/forum-v1/auth/login", "carol"),
		daveClient:  probeLogin(t, server.URL, "/forum-v1/auth/login", "dave"),
		boardSlug:   "notify",
	}

	boardPayload := env.adminClient.mustDo(http.MethodPost, "/forum-v1/admin/boards", map[string]any{
		"slug": env.boardSlug, "name": "通知测试",
	}, http.StatusOK)
	env.boardID = num(t, dataMap(t, boardPayload), "id")
	return env
}

func probeLogin(t *testing.T, base, path, username string) *testClient {
	t.Helper()
	c := newClient(t, base)
	c.mustDo(http.MethodPost, path, map[string]any{
		"username": username, "password": "password", "remember": true,
	}, http.StatusOK)
	return c
}

func mentionChip(user models.User) string {
	return fmt.Sprintf(`<span class="mention" data-tag-type="mention" data-user-id="%d">@%s</span>`, user.ID, user.Username)
}

func quoteChip(floor uint) string {
	return fmt.Sprintf(`<span class="reply-reference" data-tag-type="reply" data-floor="%d">#%d</span>`, floor, floor)
}

func (e *notificationEnv) createThread(c *testClient, title, content string) uint {
	e.t.Helper()
	payload := c.mustDo(http.MethodPost, "/forum-v1/threads", map[string]any{
		"boardId": e.boardID, "title": title, "contentHtml": content,
	}, http.StatusOK)
	return num(e.t, dataMap(e.t, payload), "id")
}

func (e *notificationEnv) createReply(c *testClient, threadID uint, content string) uint {
	e.t.Helper()
	payload := c.mustDo(http.MethodPost, fmt.Sprintf("/forum-v1/threads/%d/replies", threadID), map[string]any{
		"contentHtml": content,
	}, http.StatusOK)
	return num(e.t, dataMap(e.t, payload), "id")
}

func listNotifications(t *testing.T, c *testClient) []map[string]any {
	t.Helper()
	payload := c.mustDo(http.MethodGet, "/forum-v1/notifications?filter=all&page=1&pageSize=50", nil, http.StatusOK)
	raw := items(t, payload)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		out = append(out, item.(map[string]any))
	}
	return out
}

func notificationsForReply(t *testing.T, c *testClient, replyID uint) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, item := range listNotifications(t, c) {
		if num(t, item, "replyId") == replyID {
			out = append(out, item)
		}
	}
	return out
}

func countType(t *testing.T, items []map[string]any, typ string) int {
	t.Helper()
	count := 0
	for _, item := range items {
		if str(t, item, "type") == typ {
			count++
		}
	}
	return count
}

func loadNotification(t *testing.T, id uint) forummodels.Notification {
	t.Helper()
	var row forummodels.Notification
	if err := store.DB.First(&row, id).Error; err != nil {
		t.Fatalf("load notification %d: %v", id, err)
	}
	return row
}

func TestNotificationQuotePriorityAndEdit(t *testing.T) {
	env := setupNotificationEnv(t)

	threadID := env.createThread(env.aliceClient, "引用测试", "<p>正文</p>")
	carolFloor1 := env.createReply(env.carolClient, threadID, "<p>carol 1</p>")
	daveFloor2 := env.createReply(env.daveClient, threadID, "<p>dave 2</p>")
	carolFloor3 := env.createReply(env.carolClient, threadID, "<p>carol 3</p>")

	quotedContent := "<p>" + quoteChip(1) + " " + quoteChip(2) + " " + quoteChip(3) +
		" " + mentionChip(env.alice) + " " + mentionChip(env.dave) + "</p>"
	bobReply := env.createReply(env.bobClient, threadID, quotedContent)

	// 引用优先：Carol 只收到一条 quote（第一个楼层 #1），Dave 只收到 quote #2；
	// Alice 作为主题作者收到 reply；三人都不再收到 mention。
	carolItems := notificationsForReply(t, env.carolClient, bobReply)
	if got := len(carolItems); got != 1 || str(t, carolItems[0], "type") != "quote" || num(t, carolItems[0], "quotedFloor") != 1 {
		t.Fatalf("carol quote notifications = %v", carolItems)
	}
	daveItems := notificationsForReply(t, env.daveClient, bobReply)
	if got := len(daveItems); got != 1 || str(t, daveItems[0], "type") != "quote" || num(t, daveItems[0], "quotedFloor") != 2 {
		t.Fatalf("dave quote notifications = %v", daveItems)
	}
	aliceItems := notificationsForReply(t, env.aliceClient, bobReply)
	if got := len(aliceItems); got != 1 || str(t, aliceItems[0], "type") != "reply" {
		t.Fatalf("alice notifications = %v", aliceItems)
	}

	// 编辑新增 @ 已被 quote 覆盖的人：不重复。
	env.bobClient.mustDo(http.MethodPatch, fmt.Sprintf("/forum-v1/replies/%d", bobReply), map[string]any{
		"contentHtml": quotedContent + "<p>" + mentionChip(env.carol) + "</p>",
	}, http.StatusOK)
	if got := len(notificationsForReply(t, env.carolClient, bobReply)); got != 1 {
		t.Fatalf("carol notifications after edit = %d, want 1", got)
	}

	// Alice 新增一条楼层，Bob 编辑为引用它：Alice 已有 reply 通知，不再补 quote。
	aliceFloor4 := env.createReply(env.aliceClient, threadID, "<p>alice 4</p>")
	env.bobClient.mustDo(http.MethodPatch, fmt.Sprintf("/forum-v1/replies/%d", bobReply), map[string]any{
		"contentHtml": quotedContent + " " + quoteChip(4),
	}, http.StatusOK)
	aliceItems = notificationsForReply(t, env.aliceClient, bobReply)
	if got := len(aliceItems); got != 1 || str(t, aliceItems[0], "type") != "reply" {
		t.Fatalf("alice notifications after quote edit = %v", aliceItems)
	}

	// 新建一条无接收者的回复，再编辑引用 Dave 的楼层 #2：补发 quote。
	emptyReply := env.createReply(env.bobClient, threadID, "<p>稍后引用</p>")
	if got := len(notificationsForReply(t, env.daveClient, emptyReply)); got != 0 {
		t.Fatalf("dave notifications before edit = %d, want 0", got)
	}
	env.bobClient.mustDo(http.MethodPatch, fmt.Sprintf("/forum-v1/replies/%d", emptyReply), map[string]any{
		"contentHtml": "<p>" + quoteChip(2) + " 引用</p>",
	}, http.StatusOK)
	daveItems = notificationsForReply(t, env.daveClient, emptyReply)
	if got := len(daveItems); got != 1 || str(t, daveItems[0], "type") != "quote" || num(t, daveItems[0], "quotedFloor") != 2 {
		t.Fatalf("dave notifications after quote edit = %v", daveItems)
	}

	// 重复保存不重复发送。
	env.bobClient.mustDo(http.MethodPatch, fmt.Sprintf("/forum-v1/replies/%d", emptyReply), map[string]any{
		"contentHtml": "<p>" + quoteChip(2) + " 再次保存</p>",
	}, http.StatusOK)
	if got := len(notificationsForReply(t, env.daveClient, emptyReply)); got != 1 {
		t.Fatalf("dave notifications after repeat edit = %d, want 1", got)
	}

	// 引用主题作者（Carol 自己开主题并回复 1 楼）只发 quote，不发 reply。
	carolThread := env.createThread(env.carolClient, "Carol 主题", "<p>正文</p>")
	carolFloor := env.createReply(env.carolClient, carolThread, "<p>carol 自己的 1 楼</p>")
	quoteCarol := env.createReply(env.bobClient, carolThread, "<p>"+quoteChip(1)+" "+mentionChip(env.carol)+"</p>")
	carolOwn := notificationsForReply(t, env.carolClient, quoteCarol)
	if got := len(carolOwn); got != 1 || str(t, carolOwn[0], "type") != "quote" || num(t, carolOwn[0], "quotedFloor") != 1 {
		t.Fatalf("carol self-thread notifications = %v", carolOwn)
	}
	_ = carolFloor

	// 引用已删除楼层：不通知该楼层作者。
	env.carolClient.mustDo(http.MethodDelete, fmt.Sprintf("/forum-v1/replies/%d?reason=test", carolFloor3), nil, http.StatusOK)
	deletedQuoteReply := env.createReply(env.bobClient, threadID, "<p>"+quoteChip(3)+" 引用了已删除楼层</p>")
	if got := len(notificationsForReply(t, env.carolClient, deletedQuoteReply)); got != 0 {
		t.Fatalf("carol notifications for deleted floor = %v", got)
	}

	// 删除引用目标不删除历史通知。
	if got := len(notificationsForReply(t, env.carolClient, bobReply)); got != 1 {
		t.Fatalf("carol historical quote after floor delete = %d, want 1", got)
	}
	_ = carolFloor1
	_ = daveFloor2
	_ = aliceFloor4
}

func TestNotificationLikeCooldown(t *testing.T) {
	env := setupNotificationEnv(t)
	threadID := env.createThread(env.aliceClient, "点赞冷却", "<p>正文</p>")

	// 首次点赞 -> Alice 收到 like_thread。
	env.bobClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/threads/%d/like", threadID), nil, http.StatusOK)
	items := notificationsForReply(t, env.aliceClient, 0)
	var likeItem map[string]any
	for _, item := range items {
		if str(t, item, "type") == "like_thread" {
			likeItem = item
			break
		}
	}
	if likeItem == nil {
		t.Fatalf("alice like_thread notification missing: %v", items)
	}
	id := num(t, likeItem, "id")
	before := loadNotification(t, id)
	if before.ReadAt != nil {
		t.Fatalf("new like notification should be unread: %v", before)
	}

	// 取消点赞：通知保留。
	env.bobClient.mustDo(http.MethodDelete, fmt.Sprintf("/forum-v1/threads/%d/like", threadID), nil, http.StatusOK)
	afterUnlike := loadNotification(t, id)
	if afterUnlike.ID != id {
		t.Fatalf("notification deleted on unlike")
	}

	// 立即重新点赞：最近未读，抑制，不更新时间。
	env.bobClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/threads/%d/like", threadID), nil, http.StatusOK)
	suppressed := loadNotification(t, id)
	if !suppressed.CreatedAt.Equal(afterUnlike.CreatedAt) || suppressed.ReadAt != nil {
		t.Fatalf("cooldown not suppressed: before=%v after=%v", afterUnlike.CreatedAt, suppressed.CreatedAt)
	}
	if got := countType(t, notificationsForReply(t, env.aliceClient, 0), "like_thread"); got != 1 {
		t.Fatalf("like_thread count = %d, want 1", got)
	}

	// 标记已读后取消再赞：同一条刷新为未读并置顶。
	env.aliceClient.mustDo(http.MethodPost, "/forum-v1/notifications/read", map[string]any{
		"ids": []uint{id},
	}, http.StatusOK)
	readRow := loadNotification(t, id)
	if readRow.ReadAt == nil {
		t.Fatalf("notification not marked read")
	}
	env.bobClient.mustDo(http.MethodDelete, fmt.Sprintf("/forum-v1/threads/%d/like", threadID), nil, http.StatusOK)
	env.bobClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/threads/%d/like", threadID), nil, http.StatusOK)
	refreshed := loadNotification(t, id)
	if refreshed.ReadAt != nil || !refreshed.CreatedAt.After(readRow.CreatedAt) {
		t.Fatalf("read notification not refreshed: read=%v created=%v", readRow.CreatedAt, refreshed.CreatedAt)
	}

	// 超过窗口的未读通知：取消再赞时刷新。
	old := time.Now().Add(-11 * time.Minute)
	if err := store.DB.Model(&forummodels.Notification{}).Where("id = ?", id).
		Updates(map[string]any{"created_at": old, "read_at": nil}).Error; err != nil {
		t.Fatalf("backdate notification: %v", err)
	}
	env.bobClient.mustDo(http.MethodDelete, fmt.Sprintf("/forum-v1/threads/%d/like", threadID), nil, http.StatusOK)
	env.bobClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/threads/%d/like", threadID), nil, http.StatusOK)
	afterWindow := loadNotification(t, id)
	if afterWindow.ReadAt != nil || !afterWindow.CreatedAt.After(old) {
		t.Fatalf("expired notification not refreshed: created=%v", afterWindow.CreatedAt)
	}

	// 主页点赞：同样保留通知并应用冷却。
	env.bobClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/users/%d/like", env.alice.ID), nil, http.StatusOK)
	var profileLike *forummodels.Notification
	for _, item := range listNotifications(t, env.aliceClient) {
		if str(t, item, "type") == "like_user" {
			row := loadNotification(t, num(t, item, "id"))
			profileLike = &row
			break
		}
	}
	if profileLike == nil || profileLike.ThreadID != 0 || profileLike.ReplyID != 0 {
		t.Fatalf("like_user notification missing or malformed: %v", profileLike)
	}
	env.bobClient.mustDo(http.MethodDelete, fmt.Sprintf("/forum-v1/users/%d/like", env.alice.ID), nil, http.StatusOK)
	if kept := loadNotification(t, profileLike.ID); kept.ID != profileLike.ID {
		t.Fatalf("like_user notification deleted on unlike")
	}
	env.bobClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/users/%d/like", env.alice.ID), nil, http.StatusOK)
	if got := countType(t, listNotifications(t, env.aliceClient), "like_user"); got != 1 {
		t.Fatalf("like_user count = %d, want 1", got)
	}

	// 自赞仍然 400。
	env.aliceClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/users/%d/like", env.alice.ID), nil, http.StatusBadRequest)

	// 楼层点赞：通知保留。
	replyID := env.createReply(env.daveClient, threadID, "<p>dave 的回复</p>")
	env.bobClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/replies/%d/like", replyID), nil, http.StatusOK)
	if got := countType(t, notificationsForReply(t, env.daveClient, replyID), "like_reply"); got != 1 {
		t.Fatalf("like_reply count = %d, want 1", got)
	}
	env.bobClient.mustDo(http.MethodDelete, fmt.Sprintf("/forum-v1/replies/%d/like", replyID), nil, http.StatusOK)
	if got := countType(t, notificationsForReply(t, env.daveClient, replyID), "like_reply"); got != 1 {
		t.Fatalf("like_reply removed on unlike")
	}
}

func TestNotificationProfileAndBoard(t *testing.T) {
	env := setupNotificationEnv(t)

	// Dave 与 Carol 关注板块。
	env.daveClient.mustDo(http.MethodPut, "/forum-v1/boards/"+env.boardSlug+"/follow", nil, http.StatusOK)
	env.carolClient.mustDo(http.MethodPut, "/forum-v1/boards/"+env.boardSlug+"/follow", nil, http.StatusOK)

	// Alice 发主题并提及 Dave：Dave 只收 mention，Carol 收 board_thread。
	threadWithMention := env.createThread(env.aliceClient, "关注板块的新帖", "<p>"+mentionChip(env.dave)+"</p>")
	if got := countType(t, listNotifications(t, env.daveClient), "board_thread"); got != 0 {
		t.Fatalf("mentioned follower received board_thread: %d", got)
	}
	if got := countType(t, listNotifications(t, env.daveClient), "mention"); got != 1 {
		t.Fatalf("dave mention count = %d, want 1", got)
	}
	boardItems := listNotifications(t, env.carolClient)
	found := false
	for _, item := range boardItems {
		if str(t, item, "type") == "board_thread" && num(t, item, "threadId") == threadWithMention {
			found = true
		}
	}
	if !found {
		t.Fatalf("carol board_thread notification missing: %v", boardItems)
	}
	if got := countType(t, listNotifications(t, env.aliceClient), "board_thread"); got != 0 {
		t.Fatalf("author received own board_thread: %d", got)
	}

	// Carol 已读已有的 board_thread，方便统计后续数量。
	env.carolClient.mustDo(http.MethodPost, "/forum-v1/notifications/read", map[string]any{"all": true}, http.StatusOK)

	// Dave 取消关注后，新帖不再通知。
	env.daveClient.mustDo(http.MethodDelete, "/forum-v1/boards/"+env.boardSlug+"/follow", nil, http.StatusOK)
	daveBefore := countType(t, listNotifications(t, env.daveClient), "board_thread")
	env.createThread(env.aliceClient, "取消关注后的新帖", "<p>无提及</p>")
	if got := countType(t, listNotifications(t, env.daveClient), "board_thread"); got != daveBefore {
		t.Fatalf("dave board_thread after unfollow = %d, want %d", got, daveBefore)
	}

	// Carol 仍关注，继续收到 board_thread；历史通知保留，因此总数为 2。
	if got := countType(t, listNotifications(t, env.carolClient), "board_thread"); got != 2 {
		t.Fatalf("carol board_thread after second thread = %d, want 2", got)
	}
}

func TestNotificationPersistenceAfterDelete(t *testing.T) {
	env := setupNotificationEnv(t)
	threadID := env.createThread(env.aliceClient, "删除保留通知", "<p>正文</p>")
	replyID := env.createReply(env.bobClient, threadID, "<p>"+mentionChip(env.alice)+"</p>")

	// Alice 作为主题作者收到 reply，作为被提及者不会再收 mention；Bob 的回复没有额外通知。
	if got := countType(t, notificationsForReply(t, env.aliceClient, replyID), "reply"); got != 1 {
		t.Fatalf("alice reply notification = %d, want 1", got)
	}
	if got := countType(t, notificationsForReply(t, env.aliceClient, replyID), "mention"); got != 0 {
		t.Fatalf("alice duplicate mention = %d, want 0", got)
	}

	// 点赞回复后删除回复：通知保留。
	env.carolClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/replies/%d/like", replyID), nil, http.StatusOK)
	if got := countType(t, notificationsForReply(t, env.bobClient, replyID), "like_reply"); got != 1 {
		t.Fatalf("bob like_reply = %d, want 1", got)
	}
	env.bobClient.mustDo(http.MethodDelete, fmt.Sprintf("/forum-v1/replies/%d?reason=test", replyID), nil, http.StatusOK)
	if got := countType(t, notificationsForReply(t, env.bobClient, replyID), "like_reply"); got != 1 {
		t.Fatalf("like_reply after reply delete = %d, want 1", got)
	}

	// 删除主题：通知保留，标题仍可读。
	env.aliceClient.mustDo(http.MethodDelete, fmt.Sprintf("/forum-v1/threads/%d?reason=test", threadID), nil, http.StatusOK)
	remaining := notificationsForReply(t, env.aliceClient, replyID)
	if len(remaining) == 0 {
		t.Fatalf("reply notification removed with thread")
	}
	if str(t, remaining[0], "threadTitle") != "删除保留通知" {
		t.Fatalf("thread title after delete = %q", str(t, remaining[0], "threadTitle"))
	}

	// 主题删除后链接失效是预期；通知接口仍返回。
	env.aliceClient.mustDo(http.MethodGet, fmt.Sprintf("/forum-v1/threads/%d", threadID), nil, http.StatusNotFound)
}

type streamEvent struct {
	Name string
	Data map[string]any
}

type eventStream struct {
	t      *testing.T
	resp   *http.Response
	cancel context.CancelFunc
	events chan streamEvent
}

func openEventStream(t *testing.T, c *testClient) *eventStream {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/forum-v1/events", nil)
	if err != nil {
		cancel()
		t.Fatalf("new stream request: %v", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		cancel()
		t.Fatalf("open stream: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		cancel()
		_ = resp.Body.Close()
		t.Fatalf("stream status = %d", resp.StatusCode)
	}

	s := &eventStream{
		t:      t,
		resp:   resp,
		cancel: cancel,
		events: make(chan streamEvent, 64),
	}
	go s.read()
	t.Cleanup(s.close)
	return s
}

func (s *eventStream) read() {
	defer close(s.events)

	scanner := bufio.NewScanner(s.resp.Body)
	var name, data string
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			if name != "" {
				var payload map[string]any
				if data != "" {
					_ = json.Unmarshal([]byte(data), &payload)
				}
				s.events <- streamEvent{Name: name, Data: payload}
			}
			name, data = "", ""
		case strings.HasPrefix(line, ":"):
			continue
		case strings.HasPrefix(line, "event:"):
			name = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
	// 测试主动关闭连接时 scanner 会返回错误，这里只消费不处理。
	_ = scanner.Err()
}

func (s *eventStream) close() {
	s.cancel()
	_ = s.resp.Body.Close()
}

func (s *eventStream) next(timeout time.Duration) streamEvent {
	s.t.Helper()
	select {
	case event, ok := <-s.events:
		if !ok {
			s.t.Fatalf("event stream closed")
		}
		return event
	case <-time.After(timeout):
		s.t.Fatalf("timed out waiting for event")
		return streamEvent{}
	}
}

func (s *eventStream) expectNone(timeout time.Duration) {
	s.t.Helper()
	select {
	case event, ok := <-s.events:
		if ok {
			s.t.Fatalf("unexpected event %q: %v", event.Name, event.Data)
		}
	case <-time.After(timeout):
	}
}

func TestNotificationSSE(t *testing.T) {
	env := setupNotificationEnv(t)
	threadID := env.createThread(env.aliceClient, "SSE 测试", "<p>正文</p>")
	threadLikeCount := 0

	aliceStream := openEventStream(t, env.aliceClient)
	carolStream := openEventStream(t, env.carolClient)

	ready := aliceStream.next(time.Second)
	if ready.Name != "ready" {
		t.Fatalf("first event = %q, want ready", ready.Name)
	}
	if _, ok := ready.Data["unreadCount"]; !ok {
		t.Fatalf("ready payload = %v", ready.Data)
	}
	carolReady := carolStream.next(time.Second)
	if carolReady.Name != "ready" {
		t.Fatalf("carol first event = %q, want ready", carolReady.Name)
	}

	// Bob 点赞 Alice 的主题：Alice 收到 thread.like + notification，Carol 只收到 thread.like。
	env.bobClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/threads/%d/like", threadID), nil, http.StatusOK)
	threadLikeCount++

	aliceLike := aliceStream.next(time.Second)
	if aliceLike.Name != "thread.like" {
		t.Fatalf("alice first event = %q, want thread.like", aliceLike.Name)
	}
	if num(t, aliceLike.Data, "threadId") != threadID || num(t, aliceLike.Data, "likeCount") != uint(threadLikeCount) {
		t.Fatalf("thread.like payload = %v", aliceLike.Data)
	}
	aliceNotification := aliceStream.next(time.Second)
	if aliceNotification.Name != "notification" {
		t.Fatalf("alice second event = %q, want notification", aliceNotification.Name)
	}
	notification, ok := aliceNotification.Data["notification"].(map[string]any)
	if !ok || str(t, notification, "type") != "like_thread" {
		t.Fatalf("notification payload = %v", aliceNotification.Data)
	}
	if got := int(aliceNotification.Data["unreadCount"].(float64)); got < 1 {
		t.Fatalf("unreadCount = %d", got)
	}

	carolLike := carolStream.next(time.Second)
	if carolLike.Name != "thread.like" {
		t.Fatalf("carol first event = %q, want thread.like", carolLike.Name)
	}
	carolStream.expectNone(250 * time.Millisecond)

	// Alice 标记已读：收到 notification.read。
	notificationID := num(t, notification, "id")
	env.aliceClient.mustDo(http.MethodPost, "/forum-v1/notifications/read", map[string]any{
		"ids": []uint{notificationID},
	}, http.StatusOK)
	readEvent := aliceStream.next(time.Second)
	if readEvent.Name != "notification.read" {
		t.Fatalf("read event = %q, want notification.read", readEvent.Name)
	}
	if ids, ok := readEvent.Data["ids"].([]any); !ok || len(ids) != 1 || uint(ids[0].(float64)) != notificationID {
		t.Fatalf("read payload = %v", readEvent.Data)
	}
	carolStream.expectNone(250 * time.Millisecond)

	// Bob 点赞 Alice 的主页：Alice 收到 user.like + notification。
	env.bobClient.mustDo(http.MethodPut, fmt.Sprintf("/forum-v1/users/%d/like", env.alice.ID), nil, http.StatusOK)
	aliceUserLike := aliceStream.next(time.Second)
	if aliceUserLike.Name != "user.like" {
		t.Fatalf("alice event = %q, want user.like", aliceUserLike.Name)
	}
	aliceProfileNotification := aliceStream.next(time.Second)
	if aliceProfileNotification.Name != "notification" {
		t.Fatalf("alice event = %q, want notification", aliceProfileNotification.Name)
	}
	profileNotification, _ := aliceProfileNotification.Data["notification"].(map[string]any)
	if str(t, profileNotification, "type") != "like_user" {
		t.Fatalf("profile notification = %v", profileNotification)
	}
	carolUserLike := carolStream.next(time.Second)
	if carolUserLike.Name != "user.like" {
		t.Fatalf("carol event = %q, want user.like", carolUserLike.Name)
	}

	// Bob 关注板块：两条流都收到 board.follow。
	env.bobClient.mustDo(http.MethodPut, "/forum-v1/boards/"+env.boardSlug+"/follow", nil, http.StatusOK)
	aliceFollow := aliceStream.next(time.Second)
	if aliceFollow.Name != "board.follow" {
		t.Fatalf("alice event = %q, want board.follow", aliceFollow.Name)
	}
	carolFollow := carolStream.next(time.Second)
	if carolFollow.Name != "board.follow" {
		t.Fatalf("carol event = %q, want board.follow", carolFollow.Name)
	}
	if str(t, aliceFollow.Data, "boardSlug") != env.boardSlug {
		t.Fatalf("board.follow payload = %v", aliceFollow.Data)
	}

	// 未登录不能建立 SSE。
	anonymous := newClient(t, env.server.URL)
	anonymous.mustDo(http.MethodGet, "/forum-v1/events", nil, http.StatusUnauthorized)
}
