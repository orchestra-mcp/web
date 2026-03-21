package handlers

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// CommunityHandler handles public community profiles and posts.
type CommunityHandler struct {
	db  *gorm.DB
	hub *hub.Hub
}

// NewCommunityHandler creates a new CommunityHandler.
func NewCommunityHandler(db *gorm.DB, wsHub ...*hub.Hub) *CommunityHandler {
	h := &CommunityHandler{db: db}
	if len(wsHub) > 0 {
		h.hub = wsHub[0]
	}
	return h
}

// memberRow is the public-facing shape for a community member.
type memberRow struct {
	ID          uint              `json:"id"`
	Name        string            `json:"name"`
	Handle      string            `json:"handle"`
	Bio         string            `json:"bio"`
	AvatarURL   string            `json:"avatar_url"`
	CoverURL    string            `json:"cover_url"`
	Role        string            `json:"role"`
	SocialLinks map[string]string `json:"social_links"`
	IsPublic    bool              `json:"is_public"`
	JoinedAt    time.Time         `json:"joined_at"`
	PostCount   int64             `json:"post_count"`
}

func (h *CommunityHandler) toMemberRow(u models.User) memberRow {
	handle := settingsHandle(u)
	bio := listUserSettings(json.RawMessage(u.Settings), "bio")
	coverURL := listUserSettings(json.RawMessage(u.Settings), "cover_url")
	role := listUserSettings(json.RawMessage(u.Settings), "role")
	if role == "" {
		role = "Member"
	}

	socialLinksRaw := settingsMap(json.RawMessage(u.Settings), "social_links")
	socialLinks := map[string]string{}
	for k, v := range socialLinksRaw {
		if s, ok := v.(string); ok {
			socialLinks[k] = s
		}
	}

	var postCount int64
	h.db.Model(&models.CommunityPost{}).Where("user_id = ? AND status = 'published' AND parent_id IS NULL", u.ID).Count(&postCount)

	return memberRow{
		ID:          u.ID,
		Name:        u.Name,
		Handle:      handle,
		Bio:         bio,
		AvatarURL:   u.AvatarURL,
		CoverURL:    coverURL,
		Role:        role,
		SocialLinks: socialLinks,
		IsPublic:    true,
		JoinedAt:    u.CreatedAt,
		PostCount:   postCount,
	}
}

// ─── Public Endpoints (no auth) ─────────────────────────────────

// ListMembers handles GET /api/public/community/members
func (h *CommunityHandler) ListMembers(c fiber.Ctx) error {
	search := c.Query("search", "")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit := 30
	offset := (page - 1) * limit

	query := h.db.Where("settings->>'public_profile_enabled' = 'true'")
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(LOWER(name) LIKE LOWER(?) OR LOWER(settings->>'handle') LIKE LOWER(?))", like, like)
	}

	var total int64
	query.Model(&models.User{}).Count(&total)

	var users []models.User
	query.Order("created_at desc").Offset(offset).Limit(limit).Find(&users)

	rows := make([]memberRow, 0, len(users))
	for _, u := range users {
		rows = append(rows, h.toMemberRow(u))
	}

	return c.JSON(fiber.Map{
		"members": rows,
		"total":   total,
	})
}

// MemberProfile handles GET /api/public/community/members/:handle
// If the requester is the profile owner (via optional auth), returns the profile
// even when public_profile_enabled is false.
func (h *CommunityHandler) MemberProfile(c fiber.Ctx) error {
	handle := c.Params("handle")

	// First try: public profile lookup
	var user models.User
	err := h.db.Where("settings->>'handle' = ? AND settings->>'public_profile_enabled' = 'true'", handle).
		First(&user).Error
	if err != nil {
		// Check if the requester is the profile owner (optional auth on public route)
		caller := middleware.CurrentUser(c)
		if caller == nil {
			// Try optional auth: extract user from Authorization header even on public routes
			caller = middleware.OptionalCurrentUser(c, h.db)
		}
		if caller != nil {
			// Try to find the user by handle regardless of privacy
			if err2 := h.db.Where("settings->>'handle' = ?", handle).First(&user).Error; err2 == nil {
				if user.ID == caller.ID {
					// Owner viewing their own private profile — allow
					goto profileFound
				}
			}
		}
		// Not the owner or user not found — check if the profile exists but is private
		var privateUser models.User
		if h.db.Where("settings->>'handle' = ?", handle).First(&privateUser).Error == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "profile is private"})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "member not found"})
	}

profileFound:

	// Determine if viewer is the profile owner (privacy only applies to non-owners)
	isOwner := false
	viewer := middleware.CurrentUser(c)
	if viewer == nil {
		viewer = middleware.OptionalCurrentUser(c, h.db)
	}
	if viewer != nil && viewer.ID == user.ID {
		isOwner = true
	}

	row := h.toMemberRow(user)

	// Build social_links — try array format first (saved by profile settings page),
	// fall back to map format (legacy).
	type socialLink struct {
		Platform string `json:"platform"`
		URL      string `json:"url"`
	}
	var socialArr []socialLink
	rawLinks := settingsRaw(json.RawMessage(user.Settings), "social_links")
	if arr, ok := rawLinks.([]interface{}); ok {
		// Array format: [{platform, url}, ...]
		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				p, _ := m["platform"].(string)
				u, _ := m["url"].(string)
				if u != "" {
					socialArr = append(socialArr, socialLink{Platform: p, URL: u})
				}
			}
		}
	} else {
		// Legacy map format: {github: "https://...", ...}
		for platform, url := range row.SocialLinks {
			if url != "" {
				socialArr = append(socialArr, socialLink{Platform: platform, URL: url})
			}
		}
	}
	if socialArr == nil {
		socialArr = []socialLink{}
	}

	// Count posts and compute stats
	var postCount int64
	h.db.Model(&models.CommunityPost{}).Where("user_id = ? AND status = 'published'", user.ID).Count(&postCount)

	// Profile completeness: name, handle, bio, avatar, cover
	completeness := 0
	if user.Name != "" {
		completeness += 20
	}
	if row.Handle != "" {
		completeness += 20
	}
	if row.Bio != "" {
		completeness += 20
	}
	if user.AvatarURL != "" {
		completeness += 20
	}
	if row.CoverURL != "" {
		completeness += 20
	}

	// Recent posts
	var recentPosts []models.CommunityPost
	h.db.Where("user_id = ? AND status = 'published' AND parent_id IS NULL", user.ID).
		Order("created_at desc").Limit(5).Find(&recentPosts)

	type postRow struct {
		ID            uint      `json:"id"`
		Title         string    `json:"title"`
		Content       string    `json:"content"`
		CreatedAt     time.Time `json:"created_at"`
		LikesCount    int       `json:"likes_count"`
		CommentsCount int       `json:"comments_count"`
	}
	postRows := make([]postRow, 0, len(recentPosts))
	for _, p := range recentPosts {
		postRows = append(postRows, postRow{
			ID: p.ID, Title: p.Title, Content: p.Content,
			CreatedAt: p.CreatedAt, LikesCount: p.LikesCount, CommentsCount: p.CommentsCount,
		})
	}

	// Fetch user's teams via memberships
	type teamRow struct {
		Name      string `json:"name"`
		Slug      string `json:"slug"`
		AvatarURL string `json:"avatar_url"`
		Role      string `json:"role"`
	}
	var memberships []models.Membership
	h.db.Preload("Team").Where("user_id = ?", user.ID).Find(&memberships)
	teams := make([]teamRow, 0, len(memberships))
	for _, m := range memberships {
		teams = append(teams, teamRow{
			Name:      m.Team.Name,
			Slug:      m.Team.Slug,
			AvatarURL: m.Team.AvatarURL,
			Role:      m.Role,
		})
	}

	// Sponsors and visibility from user settings
	// Owner always sees everything — privacy only applies to public viewers
	sponsors := settingsRaw(json.RawMessage(user.Settings), "sponsors")
	showBadges := isOwner || settingsBoolDefault(json.RawMessage(user.Settings), "show_badges", true)
	showWallet := isOwner || settingsBoolDefault(json.RawMessage(user.Settings), "show_wallet", true)
	showTeams := isOwner || settingsBoolDefault(json.RawMessage(user.Settings), "show_teams", true)
	showSponsors := isOwner || settingsBoolDefault(json.RawMessage(user.Settings), "show_sponsors", true)

	return c.JSON(fiber.Map{
		"profile": fiber.Map{
			"id":           row.ID,
			"name":         row.Name,
			"handle":       row.Handle,
			"avatar_url":   row.AvatarURL,
			"cover_url":    row.CoverURL,
			"bio":          row.Bio,
			"about":        listUserSettings(json.RawMessage(user.Settings), "about"),
			"role":         row.Role,
			"location":     listUserSettings(json.RawMessage(user.Settings), "location"),
			"joined_at":    row.JoinedAt,
			"social_links": socialArr,
			"teams":        func() interface{} { if showTeams { return teams } ; return []teamRow{} }(),
			"sponsors":      func() interface{} { if showSponsors { return sponsors }; return nil }(),
			"show_badges":   showBadges,
			"show_wallet":   showWallet,
			"show_teams":    showTeams,
			"show_sponsors": showSponsors,
			"stats": func() fiber.Map {
				s := fiber.Map{
					"posts":                postCount,
					"contributions":        0,
					"profile_completeness": completeness,
				}
				// Points only visible to owner, or when show_wallet is true
				if showWallet {
					var pts map[string]interface{}
					if err := json.Unmarshal(user.Settings, &pts); err == nil {
						if p, ok := pts["points"]; ok {
							s["points"] = p
						}
					}
				}
				return s
			}(),
			"recent_posts":            postRows,
			"is_verified":             user.IsVerified,
			"show_comments_on_profile": settingsBoolDefault(json.RawMessage(user.Settings), "show_comments_on_profile", true),
			"appearance":              settingsMap(json.RawMessage(user.Settings), "appearance"),
		},
	})
}

// MemberPosts handles GET /api/public/community/members/:handle/posts
func (h *CommunityHandler) MemberPosts(c fiber.Ctx) error {
	handle := c.Params("handle")

	var user models.User
	if err := h.db.Where("settings->>'handle' = ? AND settings->>'public_profile_enabled' = 'true'", handle).
		First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "member not found"})
	}

	var posts []models.CommunityPost
	h.db.Where("user_id = ? AND status = 'published' AND parent_id IS NULL", user.ID).
		Order("created_at desc").
		Find(&posts)

	type postRow struct {
		ID            uint      `json:"id"`
		AuthorID      uint      `json:"author_id"`
		AuthorName    string    `json:"author_name"`
		AuthorHandle  string    `json:"author_handle"`
		AuthorAvatar  string    `json:"author_avatar"`
		Title         string    `json:"title"`
		Content       string    `json:"content"`
		Icon          string    `json:"icon"`
		Color         string    `json:"color"`
		Media         string    `json:"media"`
		Tags          []string  `json:"tags"`
		CreatedAt     time.Time `json:"created_at"`
		LikesCount    int       `json:"likes_count"`
		CommentsCount int       `json:"comments_count"`
	}

	authorHandle := settingsHandle(user)
	rows := make([]postRow, 0, len(posts))
	for _, p := range posts {
		var tags []string
		json.Unmarshal([]byte(p.Tags), &tags)
		if tags == nil { tags = []string{} }
		rows = append(rows, postRow{
			ID: p.ID, AuthorID: user.ID, AuthorName: user.Name,
			AuthorHandle: authorHandle, AuthorAvatar: user.AvatarURL,
			Title: p.Title, Content: p.Content,
			Icon: p.Icon, Color: p.Color, Media: p.Media, Tags: tags,
			CreatedAt: p.CreatedAt,
			LikesCount: p.LikesCount, CommentsCount: p.CommentsCount,
		})
	}

	return c.JSON(fiber.Map{"posts": rows})
}

// ShowPost handles GET /api/public/community/posts/:id
func (h *CommunityHandler) ShowPost(c fiber.Ctx) error {
	id := c.Params("id")

	var post models.CommunityPost
	if err := h.db.Where("id = ? AND status = 'published'", id).
		First(&post).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
	}

	// Get author info
	var author models.User
	h.db.First(&author, post.UserID)

	return c.JSON(fiber.Map{
		"post": fiber.Map{
			"id":             post.ID,
			"author_id":      author.ID,
			"author_name":    author.Name,
			"author_handle":  settingsHandle(author),
			"author_avatar":  author.AvatarURL,
			"title":          post.Title,
			"content":        post.Content,
			"icon":           post.Icon,
			"color":          post.Color,
			"media":          post.Media,
			"created_at":     post.CreatedAt,
			"likes_count":    post.LikesCount,
			"comments_count": post.CommentsCount,
		},
	})
}

// PostComments handles GET /api/public/community/posts/:id/comments
func (h *CommunityHandler) PostComments(c fiber.Ctx) error {
	postID := c.Params("id")

	// Fetch top-level comments (direct children of the post)
	var comments []models.CommunityPost
	h.db.Where("status = 'published' AND parent_id = ?", postID).
		Order("created_at asc").
		Find(&comments)

	type replyRow struct {
		ID         uint      `json:"id"`
		Content    string    `json:"content"`
		UserID     uint      `json:"user_id"`
		UserName   string    `json:"user_name"`
		UserAvatar string    `json:"user_avatar"`
		CreatedAt  time.Time `json:"created_at"`
		ParentID   *uint     `json:"parent_id,omitempty"`
	}
	type commentRow struct {
		ID         uint       `json:"id"`
		Content    string     `json:"content"`
		UserID     uint       `json:"user_id"`
		UserName   string     `json:"user_name"`
		UserAvatar string     `json:"user_avatar"`
		CreatedAt  time.Time  `json:"created_at"`
		Replies    []replyRow `json:"replies"`
	}

	rows := make([]commentRow, 0, len(comments))
	for _, cm := range comments {
		var u models.User
		h.db.Select("id, name, avatar_url").First(&u, cm.UserID)

		// Fetch replies (1 level deep)
		var replies []models.CommunityPost
		h.db.Where("status = 'published' AND parent_id = ?", cm.ID).
			Order("created_at asc").
			Find(&replies)

		replyRows := make([]replyRow, 0, len(replies))
		for _, r := range replies {
			var ru models.User
			h.db.Select("id, name, avatar_url").First(&ru, r.UserID)
			replyRows = append(replyRows, replyRow{
				ID: r.ID, Content: r.Content, UserID: r.UserID,
				UserName: ru.Name, UserAvatar: ru.AvatarURL,
				CreatedAt: r.CreatedAt, ParentID: r.ParentID,
			})
		}

		rows = append(rows, commentRow{
			ID: cm.ID, Content: cm.Content, UserID: cm.UserID,
			UserName: u.Name, UserAvatar: u.AvatarURL,
			CreatedAt: cm.CreatedAt, Replies: replyRows,
		})
	}

	return c.JSON(fiber.Map{"comments": rows})
}

// RelatedPosts handles GET /api/public/community/posts/:id/related
func (h *CommunityHandler) RelatedPosts(c fiber.Ctx) error {
	postID := c.Params("id")

	var post models.CommunityPost
	if err := h.db.First(&post, postID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
	}

	// Related = same author's other posts + recent posts
	var related []models.CommunityPost
	h.db.Where("id != ? AND status = 'published' AND parent_id IS NULL", postID).
		Where("user_id = ?", post.UserID).
		Order("created_at desc").
		Limit(5).
		Find(&related)

	// Fill with recent posts if not enough
	if len(related) < 5 {
		remaining := 5 - len(related)
		existingIDs := []uint{post.ID}
		for _, r := range related {
			existingIDs = append(existingIDs, r.ID)
		}

		var morePosts []models.CommunityPost
		h.db.Where("id NOT IN ? AND status = 'published' AND parent_id IS NULL", existingIDs).
			Order("created_at desc").
			Limit(remaining).
			Find(&morePosts)
		related = append(related, morePosts...)
	}

	// Build response with author info
	type relatedRow struct {
		ID            uint      `json:"id"`
		AuthorName    string    `json:"author_name"`
		AuthorHandle  string    `json:"author_handle"`
		AuthorAvatar  string    `json:"author_avatar"`
		Title         string    `json:"title"`
		Content       string    `json:"content"`
		CreatedAt     time.Time `json:"created_at"`
		LikesCount    int       `json:"likes_count"`
		CommentsCount int       `json:"comments_count"`
	}

	// Cache author lookups
	authorCache := map[uint]models.User{}
	rows := make([]relatedRow, 0, len(related))
	for _, rp := range related {
		author, ok := authorCache[rp.UserID]
		if !ok {
			h.db.Select("id, name, avatar_url, settings").First(&author, rp.UserID)
			authorCache[rp.UserID] = author
		}
		rows = append(rows, relatedRow{
			ID: rp.ID, AuthorName: author.Name,
			AuthorHandle: settingsHandle(author), AuthorAvatar: author.AvatarURL,
			Title: rp.Title, Content: rp.Content, CreatedAt: rp.CreatedAt,
			LikesCount: rp.LikesCount, CommentsCount: rp.CommentsCount,
		})
	}

	return c.JSON(fiber.Map{"posts": rows})
}

// ─── Authenticated Endpoints ────────────────────────────────────

// CreatePost handles POST /api/community/posts
func (h *CommunityHandler) CreatePost(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Icon    string   `json:"icon"`
		Color   string   `json:"color"`
		Media   string   `json:"media"`
		Tags    []string `json:"tags"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Title == "" || body.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title and content are required"})
	}

	tagsJSON := "[]"
	if len(body.Tags) > 0 {
		if b, err := json.Marshal(body.Tags); err == nil {
			tagsJSON = string(b)
		}
	}

	post := models.CommunityPost{
		UserID:  user.ID,
		Title:   body.Title,
		Content: body.Content,
		Icon:    body.Icon,
		Color:   body.Color,
		Media:   body.Media,
		Tags:    tagsJSON,
		Status:  "published",
	}

	if err := h.db.Create(&post).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create post"})
	}

	broadcastSync(h.hub, user.ID, "community_post", post.ID, "upsert")
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"post": fiber.Map{
			"id":             post.ID,
			"author_id":      user.ID,
			"author_name":    user.Name,
			"author_handle":  settingsHandle(*user),
			"author_avatar":  user.AvatarURL,
			"title":          post.Title,
			"content":        post.Content,
			"icon":           post.Icon,
			"color":          post.Color,
			"media":          post.Media,
			"tags":           body.Tags,
			"created_at":     post.CreatedAt,
			"likes_count":    0,
			"comments_count": 0,
		},
	})
}

// UpdatePost handles PUT /api/community/posts/:id
func (h *CommunityHandler) UpdatePost(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
	}

	var post models.CommunityPost
	if err := h.db.First(&post, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
	}

	if post.UserID != user.ID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you can only edit your own posts"})
	}

	var body struct {
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Icon    string   `json:"icon"`
		Color   string   `json:"color"`
		Media   string   `json:"media"`
		Tags    []string `json:"tags"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Title == "" || body.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title and content are required"})
	}

	post.Title = body.Title
	post.Content = body.Content
	post.Icon = body.Icon
	post.Color = body.Color
	post.Media = body.Media
	tagsJSON := "[]"
	if len(body.Tags) > 0 {
		if b, err := json.Marshal(body.Tags); err == nil {
			tagsJSON = string(b)
		}
	}
	post.Tags = tagsJSON
	if err := h.db.Save(&post).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update post"})
	}

	broadcastSync(h.hub, user.ID, "community_post", post.ID, "upsert")
	return c.JSON(fiber.Map{
		"post": fiber.Map{
			"id":             post.ID,
			"author_id":      user.ID,
			"author_name":    user.Name,
			"author_handle":  settingsHandle(*user),
			"author_avatar":  user.AvatarURL,
			"title":          post.Title,
			"content":        post.Content,
			"icon":           post.Icon,
			"color":          post.Color,
			"media":          post.Media,
			"tags":           body.Tags,
			"created_at":     post.CreatedAt,
			"likes_count":    post.LikesCount,
			"comments_count": post.CommentsCount,
		},
	})
}

// DeletePost handles DELETE /api/community/posts/:id
func (h *CommunityHandler) DeletePost(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
	}

	var post models.CommunityPost
	if err := h.db.First(&post, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
	}

	if post.UserID != user.ID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you can only delete your own posts"})
	}

	if err := h.db.Delete(&post).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete post"})
	}

	broadcastSync(h.hub, user.ID, "community_post", post.ID, "delete")
	return c.JSON(fiber.Map{"ok": true})
}

// MyPosts handles GET /api/community/posts (authenticated — current user's feed)
func (h *CommunityHandler) MyPosts(c fiber.Ctx) error {
	var posts []models.CommunityPost
	h.db.Where("status = ? AND parent_id IS NULL", "published").Order("created_at DESC").Limit(50).Find(&posts)
	return c.JSON(fiber.Map{"posts": posts})
}

// AddComment handles POST /api/community/posts/:id/comments
func (h *CommunityHandler) AddComment(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	postID := c.Params("id")

	// Verify post exists
	var post models.CommunityPost
	if err := h.db.First(&post, postID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
	}

	var body struct {
		Content  string `json:"content"`
		ParentID *uint  `json:"parent_id,omitempty"` // reply to a comment (not the post)
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "content is required"})
	}

	// Determine parent: if replying to a comment, use that comment's ID; otherwise use the post ID
	var parentID uint
	if body.ParentID != nil && *body.ParentID > 0 {
		// Replying to a comment — verify the comment exists and belongs to this post
		var parentComment models.CommunityPost
		if err := h.db.First(&parentComment, *body.ParentID).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "parent comment not found"})
		}
		parentID = *body.ParentID
	} else {
		pid, err := strconv.ParseUint(postID, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
		}
		parentID = uint(pid)
	}

	comment := models.CommunityPost{
		UserID:   user.ID,
		Title:    "",
		Content:  body.Content,
		Status:   "published",
		ParentID: &parentID,
	}

	if err := h.db.Create(&comment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create comment"})
	}

	// Increment comment count on parent post
	h.db.Model(&post).UpdateColumn("comments_count", gorm.Expr("comments_count + 1"))

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"comment": fiber.Map{
			"id":          comment.ID,
			"content":     comment.Content,
			"user_id":     user.ID,
			"user_name":   user.Name,
			"user_avatar": user.AvatarURL,
			"created_at":  comment.CreatedAt,
		},
	})
}

// ToggleLike handles POST /api/community/posts/:id/like
func (h *CommunityHandler) ToggleLike(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	postID := c.Params("id")

	// Verify post exists
	var post models.CommunityPost
	if err := h.db.First(&post, postID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
	}

	// Check if already liked
	var existingLike models.CommunityLike
	err := h.db.Where("user_id = ? AND post_id = ?", user.ID, post.ID).First(&existingLike).Error

	if err == nil {
		// Already liked — remove
		h.db.Delete(&existingLike)
		h.db.Model(&post).UpdateColumn("likes_count", gorm.Expr("GREATEST(likes_count - 1, 0)"))
		return c.JSON(fiber.Map{"liked": false, "likes_count": post.LikesCount - 1})
	}

	// Not liked — add
	like := models.CommunityLike{
		UserID: user.ID,
		PostID: post.ID,
	}
	h.db.Create(&like)
	h.db.Model(&post).UpdateColumn("likes_count", gorm.Expr("likes_count + 1"))

	return c.JSON(fiber.Map{"liked": true, "likes_count": post.LikesCount + 1})
}

// MemberActivity handles GET /api/public/community/members/:handle/activity
// Returns a timeline of the user's public posts and comments.
func (h *CommunityHandler) MemberActivity(c fiber.Ctx) error {
	handle := c.Params("handle")
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit > 50 {
		limit = 50
	}

	// Resolve user by handle (stored in settings JSON).
	var user models.User
	if err := h.db.Where("settings->>'handle' = ?", handle).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "member not found"})
	}

	// Check if profile is public.
	isPublic := listUserSettings(json.RawMessage(user.Settings), "public_profile_enabled")
	if isPublic != "true" {
		// Allow if the viewer is the owner.
		viewer := middleware.CurrentUser(c)
		if viewer == nil || viewer.ID != user.ID {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "profile is private"})
		}
	}

	type activityItem struct {
		Type       string    `json:"type"`        // "post", "comment", "shared_note", "shared_skill", etc.
		ID         uint      `json:"id"`
		Title      string    `json:"title"`
		Excerpt    string    `json:"excerpt"`
		ParentID   *uint     `json:"parent_id,omitempty"`
		EntityType string    `json:"entity_type,omitempty"` // for shared content
		Slug       string    `json:"slug,omitempty"`        // for shared content URLs
		CreatedAt  time.Time `json:"created_at"`
	}

	var items []activityItem

	// Fetch published posts.
	var posts []models.CommunityPost
	h.db.Where("user_id = ? AND status = 'published' AND parent_id IS NULL", user.ID).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&posts)

	for _, p := range posts {
		excerpt := p.Content
		if len(excerpt) > 200 {
			excerpt = excerpt[:200] + "..."
		}
		items = append(items, activityItem{
			Type:      "post",
			ID:        p.ID,
			Title:     p.Title,
			Excerpt:   excerpt,
			CreatedAt: p.CreatedAt,
		})
	}

	// Fetch comments (posts with parent_id) — only if user allows it.
	showComments := listUserSettings(json.RawMessage(user.Settings), "show_comments_on_profile")
	var comments []models.CommunityPost
	if showComments != "false" {
		h.db.Where("user_id = ? AND status = 'published' AND parent_id IS NOT NULL", user.ID).
			Order("created_at desc").
			Limit(limit).
			Offset(offset).
			Find(&comments)
	}

	for _, cm := range comments {
		excerpt := cm.Content
		if len(excerpt) > 200 {
			excerpt = excerpt[:200] + "..."
		}
		items = append(items, activityItem{
			Type:      "comment",
			ID:        cm.ID,
			Excerpt:   excerpt,
			ParentID:  cm.ParentID,
			CreatedAt: cm.CreatedAt,
		})
	}

	// Fetch public shared content (notes, skills, agents, workflows).
	var shares []models.SharedContent
	h.db.Where("user_id = ? AND visibility = 'public'", user.ID).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&shares)

	for _, s := range shares {
		excerpt := s.Description
		if excerpt == "" {
			excerpt = s.Content
		}
		if len(excerpt) > 200 {
			excerpt = excerpt[:200] + "..."
		}
		items = append(items, activityItem{
			Type:       "shared_" + s.EntityType,
			ID:         s.ID,
			Title:      s.Title,
			Excerpt:    excerpt,
			EntityType: s.EntityType,
			Slug:       s.Slug,
			CreatedAt:  s.CreatedAt,
		})
	}

	return c.JSON(fiber.Map{
		"activity": items,
		"count":    len(items),
		"handle":   handle,
	})
}
