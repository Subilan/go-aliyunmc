# Moon 论坛前台实施计划（moon-client-web）

> 状态：待评审  
> 适用范围：论坛前台 `moon-client-web`  
> 前置条件：后端任务（`moon-plan.md`）已完成，单元测试与 API 级 E2E 通过，API 契约冻结。  
> 本文件对应独立前端会话；后端 API、数据模型、权限与内容契约以 `moon-plan.md` 为准，本会话不修改后端契约。  
> 管理后台前端不在本文件范围。
>
> 建议阅读顺序：0 实施不变式 → 1 技术栈 → 2 能力范围 → 3 数据层 → 4 编辑器与提及 → 5 页面职责 → 8 实施阶段。

---

## 0. 实施不变式（硬性）

> 本期前端实施的硬性约束：**界面样式代码冻结**。只做数据接入与 API 接线，不改视觉。

### 0.1 禁止修改

- 任何 CSS / Tailwind / 主题 / 设计 token，包括 `src/index.css`、`tailwind.config.*`、字体、颜色、间距、圆角、阴影、动画、断点等。
- 组件的视觉呈现：`className`、`style`、`variant` / `color` / `size` / `radius` 等样式 props、布局与 DOM 层级、图标、排版。
- 现有 UI 组件的结构与视觉行为；不得为了接数据重排 JSX 层级或替换 UI 组件。
- 不得“顺手”修复样式问题；发现样式缺陷时单独记录并另开任务，不在本期修改。

### 0.2 允许修改

- 类型定义与 DTO 映射。
- API client、请求函数、query keys、缓存、乐观更新、轮询。
- `AuthProvider`、session 与权限状态。
- 数据来源 props、事件处理中的 API 调用、loading / error / empty 状态的数据接线。
- 新增数据 hooks / adapters；删除本地 seed / mock / localStorage 业务数据。
- 在既有组件与样式内新增数据驱动的展示分支或文案（例如新的通知类型）；不得新增或修改视觉样式。
- 因接入真实数据必须改动 JSX 时，只允许改动数据绑定和条件渲染的数据来源；除真实数据内容变化外，渲染出的视觉结果必须保持不变。

### 0.3 判定与评审

- 出现以下内容的 diff 直接拒绝：CSS 文件/配置、`className`、`style`、样式 props、主题/布局/图标改动。
- 允许的 diff 只应包含：类型、API 调用、hooks、状态管理、数据映射、数据源替换、必要的数据绑定。
- 代码评审第一检查项是“是否改变界面视觉”；无法确定时默认不改。
- 若数据接入在不改样式的前提下确实无法完成，先记录冲突点和最小改动方案并请求确认，不得自行放宽。

---

## 1. 技术栈

| 领域 | 选型 | 说明 |
|---|---|---|
| 框架 | React 19 + TypeScript + Vite | 沿用现有前台 |
| UI | HeroUI + Tailwind 4 | 沿用现有设计体系 |
| 富文本 | Tiptap | 现有编辑器扩展 |
| 数据层 | `@tanstack/react-query` | 缓存、分页、乐观更新、失效重取 |
| 测试 | Playwright | 主流程 E2E |

---

## 2. 能力范围

页面数据全部通过 API 读写，前台需覆盖：

- 首页：全部非置顶主题流、置顶主题轮播、最近回复侧栏。
- 板块：板块列表、板块主题流、关注板块、板块置顶。
- 主题：主题详情、跟帖、楼层目录、置顶/锁定、点赞、编辑/删除自己的内容。
- 跟帖：富文本、`re#楼层` 引用、`@提及`、楼层号、跟帖点赞、编辑/删除。
- 用户：个人主页、bio、头像裁剪、头像形状、主页背景、用户点赞、主题/回复列表、获赞数。
- 通知：回复通知、@ 提及、主题点赞、跟帖点赞；未读红点、全部/未读筛选、单条/全部已读。
- 搜索：标题、正文、标签、作者、板块、回复正文的多关键词 AND 搜索；热门标签。
- 富文本：标题、列表、引用、代码、表格、图片（上传/外链、缩放、对齐）、链接、emoji、Markdown 导入/导出。
- 提及：输入 `@` 打开用户选择器，插入原子 chip，渲染为个人主页链接；被提及者收到通知。
- 登录：共享账号登录，不开放注册；退出登录。

---

## 3. 数据层

- 所有请求统一走 `src/lib/api.ts`：
  - `request<T>(path, options)`：`credentials: 'include'`，解析 `{data}`/`{details}`，抛出 `ApiError(status, message)`。
  - `API_BASE` 默认 `/forum-v1`，开发环境通过 Vite 代理转发。
- 使用 `@tanstack/react-query` 管理服务端状态，按资源划分 query keys：
  - `['site']`
  - `['boards']`, `['board', slug]`
  - `['threads', boardSlug, page]`, `['pinned', boardSlug]`
  - `['thread', id]`, `['replies', id, page]`
  - `['user', id]`, `['user-threads', id, page]`, `['user-replies', id, page]`
  - `['notifications', filter, page]`, `['unread-count']`
  - `['mention-users', q]`
  - `['search', q, page]`, `['tags', boardSlug]`
- 认证上下文只保存当前用户与站点配置，不承载业务数据：
  - 启动时 `GET /auth/me`。
  - `login(username, password)` / `logout()` 调 API。
  - 401 时打开登录弹窗；403 显示 toast。
- 页面数据全部来自 API DTO 与 query cache；组件不维护跨页面的业务实体数组。
- 点赞/关注使用乐观更新：先改 query cache，失败回滚并 toast。

---

## 4. 编辑器、提及与图片

- 输入 `@` 打开用户选择器，按 `username` / `nickname` 调 `GET /users/search?q=` 搜索；选择后插入原子 chip：`<span class="mention" data-tag-type="mention" data-user-id="3">@DuskWalker</span>`。
- chip 的展示文本始终是 `@username`，昵称只用于选择器搜索；chip 作为原子节点，退格整体删除。
- 提及 chip 导出 Markdown 时为 `@username`；正文渲染时转换为 `/user/{id}` 链接。
- 楼层引用与提及都通过 `data-tag-type` 区分，`class` 只用于样式。
- 编辑器插入图片时先并发 `uploadImage(file, 'post')`，成功后插入返回的 URL；失败提示且不插入。
- 正文禁止 `data:`/Base64 图片；粘贴时移除 data URL，只保留已上传 URL。
- 头像裁剪成 512×512，导出 WebP（0.88），上传 `kind=avatar`，再 `PUT /me/avatar` 绑定。
- 主页背景按 16:9、最长边 1920 缩放，导出 WebP（0.85），上传 `kind=background`，再 `PUT /me/background` 绑定。

---

## 5. 页面与组件职责

| 文件/模块 | 要求 |
|---|---|
| `src/types.ts` | API DTO 类型；ID 为 number；`avatarShape` 属于用户 DTO |
| `src/lib/forum-context.ts` | 仅提供 auth/site 上下文，不承载业务数据 |
| `src/lib/forum-store.tsx` | `AuthProvider` + QueryClientProvider |
| `src/lib/notifications.ts` | 基于 API 的 `useNotifications`，不做本地派生 |
| `src/lib/avatar.ts` | `getAvatarUrl` 优先 `user.avatarUrl`；形状取 `user.avatarShape` |
| `src/lib/constants.ts` | 仅保留 API base 等常量 |
| `src/components/Layout.tsx` | 站点名/页脚/公告来自 `/site`；通知轮询 |
| `src/components/LoginDialog.tsx` | 调 `/auth/login` |
| `src/components/ThreadCard.tsx` | 使用内嵌 author/board/replyCount |
| `src/components/PinnedThreads.tsx` | 调 `/pinned-threads` |
| `src/components/FeedSidebar.tsx` | 调 `/recent-replies`；发帖板块选择来自 `/boards` |
| `src/components/NotificationsPopover.tsx` | 未读数/列表来自 API；`mention` 类型显示“提到了你” |
| `src/components/editor/RichTextEditor.tsx` | `@` 提及选择器；原子 mention chip；候选来自 `/users/search` |
| `src/components/PostContent.tsx` | mention chip 渲染为 `/user/{id}` 链接；标签提取跳过 mention |
| `src/pages/HomePage.tsx` | 分页加载主题 |
| `src/pages/BoardsPage.tsx` | 板块 API；空状态引导管理员创建板块 |
| `src/pages/TopicPage.tsx` | 板块 slug、置顶、关注、发帖 |
| `src/pages/ThreadPage.tsx` | 主题/跟帖分页、稳定楼层、删除占位、`canEdit/canDelete/canModerate` |
| `src/pages/UserPage.tsx` | 用户 DTO、计数、头像/背景上传 |
| `src/pages/SearchPage.tsx` | 服务端搜索、分页、热门标签 |
| `src/pages/NotificationsPage.tsx` | 服务端通知、全部/未读、标记已读；`mention` 类型文案与跳转 |
| `src/components/ReplyComposer.tsx` | 异步创建回复；引用楼层用 `floor` |
| `src/components/EditPostDialog.tsx` | 异步保存与错误处理 |
| `src/lib/reply-reference-options.ts` | 使用 `reply.floor` |
| `vite.config.ts` | 代理 `/forum-v1` 到后端 |

---

## 6. 数据契约注意

- `Board` 同时有数字 `id` 与 `slug`；路由使用 slug，发帖使用 `boardId`。
- 通知由服务端返回，带 `id` 与 `read`；未读状态以服务端为准。
- 点赞、关注、头像形状、用户背景、标签等状态全部来自 DTO，前端不维护全局业务状态。
- 标签由后端提取并返回，前端只负责渲染和搜索链接。
- 楼层引用契约为 `span[data-tag-type="reply"][data-floor]`，提及契约为 `span[data-tag-type="mention"][data-user-id]`；`class` 只用于样式，前端与服务端都不得用 `class` 区分两种 chip。
- mention chip 的文本为 `@username`；服务端写入时会重新校验用户并规范化文本。

---

## 7. 前端安全

- 正文继续由 `PostContent` 在渲染前 sanitize（纵深防御），但服务端已 sanitize。
- 配置 CSP（由前端静态服务器下发）：
  - `default-src 'self'`
  - `img-src 'self' https://minotar.net https:`
  - `connect-src 'self' <API origin>`
  - `script-src 'self'`
- 不在前端存储密码/token；只依赖 HttpOnly session cookie。
- 外链图片允许但可能泄漏 referrer；如需更严格，可在 CSP 中限制 `img-src` 只允许本站，并取消外链图片功能。

---

## 8. 实施阶段

开始条件：后端 API 已完成、单元测试与 API 级 E2E 通过，且契约冻结。所有阶段均受第 0 节约束：只允许数据与 API 接线，不得修改任何界面样式代码。

### Phase 1：数据层与认证（1 个 PR）

- API client + TanStack Query + `AuthProvider`。
- 类型替换为后端 DTO；移除本地业务数据源。
- 登录/登出/`auth/me`、401/403 处理。

### Phase 2：页面接入（1~2 个 PR）

- 页面/组件按第 5 节实现。
- 主题/跟帖分页、稳定楼层、删除占位、`canEdit/canDelete/canModerate`。
- 通知轮询：每 30 秒请求 `/notifications/unread-count`，窗口聚焦时刷新；打开通知面板时按条标记已读。
- 搜索分页、热门标签。

### Phase 3：编辑器与上传（1 个 PR）

- 编辑器图片上传、头像/背景上传。
- @ 提及选择器、mention chip 渲染、Markdown 导出。
- 服务端拒绝 `data:` 图片的前端兜底（粘贴清理）。

### Phase 4：端到端验收（与后端联调）

- 主流程 E2E。
- 真实 API 上无本地假数据。

**验收**：前台全部功能在真实 API 上可用；`npm run build` / `npm run lint` 通过；diff 中无任何界面样式改动。

---

## 9. E2E 与工程检查

### 9.1 Playwright 主流程

- 测试数据通过 API 或种子夹具准备，不依赖管理后台前端。
- 普通用户登录 → 发主题（含图片）→ 发跟帖（含 `re#` 引用、`@` 提及）→ 点赞 → 通知出现（含 mention）→ 标记已读。
- 用户 A 在主题或跟帖中 @ 用户 B → 用户 B 登录后收到 mention 通知并可跳转到对应内容。
- 用户主页：改 bio → 上传头像 → 切换形状 → 上传背景。
- 搜索：关键词、标签、作者、无结果。
- 板块审核用户：授权板块内置顶/删除；其他板块无按钮/403。
- 删除回复后楼层占位与引用仍可跳转。

### 9.2 工程检查

- `npm run build`
- `npm run lint`
- 检查 diff：无 CSS / Tailwind / 主题 / `className` / `style` / 样式 props 改动
- 页面视觉与改动前一致（仅允许真实数据内容不同）；关键页面做改动前后截图对比
- 后端 API 契约变更时同步更新本文件与 DTO 类型

---

## 10. 风险与注意

| 风险 | 影响 | 缓解 |
|---|---|---|
| API 契约漂移 | 前端类型与后端 DTO 不一致 | 类型集中在 `src/types.ts`；后端变更同步更新 |
| 后端契约缺口 | 前端被迫临时改协议 | 回到后端任务补齐契约并重跑 API E2E，不在前端会话改协议 |
| 本地假数据残留 | 页面看似可用但未接后端 | 删除本地业务数据源；验收以真实 API 为准 |
| 外链图片 | 隐私/失效/混合内容 | CSP 限制 `img-src`；必要时取消外链图片 |
| 头像服务不可用 | 默认头像加载失败 | 前端加载失败回退 |
| 楼层重排 | 历史 `re#楼层` 引用错位 | 按 `floor` 渲染，不依赖数组下标 |
| 接数据时顺手改样式 | 冻结的界面视觉被破坏 | 第 0 节不变式；review 检查样式 diff；样式问题另开任务 |