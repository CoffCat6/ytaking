package blog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type SiteStore struct {
	path string
	mu   sync.RWMutex
	data SiteProfile
}

func NewSiteStore(path string) (*SiteStore, error) {
	store := &SiteStore{path: path}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *SiteStore) Get() SiteProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

func (s *SiteStore) Update(profile SiteProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = profile
	return s.save()
}

func (s *SiteStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.path); err != nil {
		if os.IsNotExist(err) {
			s.data = defaultProfile()
			return s.save()
		}
		return err
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		s.data = defaultProfile()
		return s.save()
	}

	var profile SiteProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return err
	}
	s.data = profile
	return nil
}

func (s *SiteStore) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(s.path, data, 0o644)
}

func defaultProfile() SiteProfile {
	return SiteProfile{
		Title:       "Ytakling(言说)",
		Tagline:     "记录产品、工程与创作的个人博客",
		Intro:       "我在做能长期生长的产品，喜欢清晰的结构、可维护的代码与有温度的文字。",
		Positioning: "长期主义的产品与工程作者，写作是我的思考方式。",
		Skills: []string{
			"产品策略",
			"Go 与架构",
			"写作系统",
			"个人知识管理",
		},
		Avatar:     "",
		AvatarPosX: 50,
		AvatarPosY: 50,
		AvatarScale: 1.0,
		Location:   "杭州 - CN",
		Email:      "hi@example.com",
		Newsletter: "每月一封：复盘、工具、写作。",
		CurrentFocus: []string{
			"建立一套内容发布流程",
			"输出长期产品思考",
			"记录工程实践与复盘",
		},
		SocialLinks: []SocialLink{
			{Name: "GitHub", URL: "https://github.com/"},
		},
	}
}
