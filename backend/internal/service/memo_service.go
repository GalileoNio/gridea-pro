package service

import (
	"context"
	"fmt"
	"gridea-pro/backend/internal/domain"
	"regexp"
	"sort"
	"sync"
	"time"
)

var (
	// compile once for performance
	tagRegexp = regexp.MustCompile(`#([\p{L}\p{N}_]+)`)
)

type MemoService struct {
	repo domain.MemoRepository
	mu   sync.RWMutex
}

func NewMemoService(repo domain.MemoRepository) *MemoService {
	return &MemoService{repo: repo}
}

func (s *MemoService) LoadMemos(ctx context.Context) ([]domain.Memo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	memos, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	// 按 CreatedAt 倒序——避免「用户改了发布时间但列表位置不动」的错觉。
	// 存储仍按创建插入顺序（CreateMemo 是 prepend），排序只在读时生效，
	// 保证所有消费者（前端、Dashboard、MCP）看到的次序一致。
	sort.SliceStable(memos, func(i, j int) bool {
		return memos[i].CreatedAt.After(memos[j].CreatedAt)
	})
	return memos, nil
}

func (s *MemoService) SaveMemos(ctx context.Context, memos []domain.Memo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.repo.SaveAll(ctx, memos)
}

// CreateMemo 创建闪念。createdAt 为零值时使用当前时间（默认行为），
// 非零时按调用方指定的发布时间创建——支持前端「设置发布时间」。
func (s *MemoService) CreateMemo(ctx context.Context, content string, createdAt time.Time) (*domain.Memo, error) {
	if content == "" {
		return nil, fmt.Errorf("content is empty")
	}

	now := time.Now()
	publishedAt := createdAt
	if publishedAt.IsZero() {
		publishedAt = now
	}
	newMemo := domain.Memo{
		Content:   content,
		Tags:      extractTags(content),
		Images:    []string{},
		CreatedAt: publishedAt,
		UpdatedAt: now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	memos, err := s.repo.List(ctx)
	if err != nil {
		memos = []domain.Memo{}
	}

	// Prepend
	memos = append([]domain.Memo{newMemo}, memos...)

	if err := s.repo.SaveAll(ctx, memos); err != nil {
		return nil, err
	}

	return &memos[0], nil
}

func (s *MemoService) UpdateMemo(ctx context.Context, memo domain.Memo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	memos, err := s.repo.List(ctx)
	if err != nil {
		return err
	}

	// Parse content for tags
	memo.Tags = extractTags(memo.Content)
	memo.UpdatedAt = time.Now()

	found := false
	for i := range memos {
		if memos[i].ID == memo.ID {
			memos[i].Content = memo.Content
			memos[i].Tags = memo.Tags
			memos[i].UpdatedAt = memo.UpdatedAt
			memos[i].Images = memo.Images
			// 允许编辑时修改发布时间；零值表示前端未指定，保留原值
			if !memo.CreatedAt.IsZero() {
				memos[i].CreatedAt = memo.CreatedAt
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("memo not found")
	}

	return s.repo.SaveAll(ctx, memos)
}

func (s *MemoService) DeleteMemo(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	memos, err := s.repo.List(ctx)
	if err != nil {
		return err
	}

	filtered := make([]domain.Memo, 0)
	found := false
	for _, memo := range memos {
		if memo.ID != id {
			filtered = append(filtered, memo)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("memo not found")
	}

	return s.repo.SaveAll(ctx, filtered)
}

func (s *MemoService) RenameTag(ctx context.Context, oldName, newName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	memos, err := s.repo.List(ctx)
	if err != nil {
		return err
	}

	count := 0
	updatedMemos := make([]domain.Memo, 0)

	// Pre-compile regex for replacement
	// Using QuoteMeta to safely escape the tag name
	re := regexp.MustCompile(`#` + regexp.QuoteMeta(oldName) + `([^\p{L}\p{N}_]|$)`)

	for i := range memos {
		hasTag := false
		for _, t := range memos[i].Tags {
			if t == oldName {
				hasTag = true
				break
			}
		}

		if hasTag {
			memos[i].Content = re.ReplaceAllString(memos[i].Content, "#"+newName+"$1")
			memos[i].Tags = extractTags(memos[i].Content)
			memos[i].UpdatedAt = time.Now()
			count++
		}
		updatedMemos = append(updatedMemos, memos[i])
	}

	if count > 0 {
		return s.repo.SaveAll(ctx, updatedMemos)
	}
	return nil
}

func (s *MemoService) DeleteTag(ctx context.Context, tagName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	memos, err := s.repo.List(ctx)
	if err != nil {
		return err
	}

	count := 0
	updatedMemos := make([]domain.Memo, 0)

	// Pre-compile regex for deletion
	re := regexp.MustCompile(`#` + regexp.QuoteMeta(tagName) + `([^\p{L}\p{N}_]|$)`)

	for i := range memos {
		hasTag := false
		for _, t := range memos[i].Tags {
			if t == tagName {
				hasTag = true
				break
			}
		}

		if hasTag {
			memos[i].Content = re.ReplaceAllString(memos[i].Content, tagName+"$1")
			memos[i].Tags = extractTags(memos[i].Content)
			memos[i].UpdatedAt = time.Now()
			count++
		}
		updatedMemos = append(updatedMemos, memos[i])
	}

	if count > 0 {
		return s.repo.SaveAll(ctx, updatedMemos)
	}
	return nil
}

func (s *MemoService) GetMemoStats(ctx context.Context) (*domain.MemoStats, error) {
	// Re-use LoadMemos which has Read Lock
	memos, err := s.LoadMemos(ctx)
	if err != nil {
		return nil, err
	}

	tagCount := make(map[string]int)
	for _, memo := range memos {
		for _, tag := range memo.Tags {
			tagCount[tag]++
		}
	}

	var tagStats []domain.TagStat
	for name, count := range tagCount {
		tagStats = append(tagStats, domain.TagStat{
			Name:  name,
			Count: count,
		})
	}

	sort.Slice(tagStats, func(i, j int) bool {
		return tagStats[i].Count > tagStats[j].Count
	})

	heatmap := make(map[string]int)
	now := time.Now()
	for i := 0; i < 365; i++ {
		date := now.AddDate(0, 0, -i).Format("2006-01-02") // Use direct format string or domain constant if available
		heatmap[date] = 0
	}

	for _, memo := range memos {
		// memo.CreatedAt is now time.Time, no parsing needed
		date := memo.CreatedAt.Format("2006-01-02")
		if _, exists := heatmap[date]; exists {
			heatmap[date]++
		}
	}

	return &domain.MemoStats{
		Total:   len(memos),
		Tags:    tagStats,
		Heatmap: heatmap,
	}, nil
}

// extractTags helper
func extractTags(content string) []string {
	matches := tagRegexp.FindAllStringSubmatch(content, -1)

	tagSet := make(map[string]bool)
	tags := make([]string, 0)
	for _, match := range matches {
		if len(match) > 1 {
			tag := match[1]
			if !tagSet[tag] {
				tagSet[tag] = true
				tags = append(tags, tag)
			}
		}
	}
	return tags
}
