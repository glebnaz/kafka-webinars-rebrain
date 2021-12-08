package models

import "sync"

type Post struct {
	ID        string `json:"id"`
	Payload   string `json:"payload"`
	CreatedAt int64  `json:"created_at"`
	CreatedBy int32  `json:"created_by"`
}

type User struct {
	ID         int32   `json:"id"`
	Name       string  `json:"name"`
	Subcribers []int32 `json:"subcribers"`
}

type UserStore struct {
	m sync.RWMutex

	data map[int32]User
}

func NewUsersStore() UserStore {
	d := make(map[int32]User)
	return UserStore{
		data: d,
	}
}

type PostStore struct {
	m sync.RWMutex

	data map[string]Post
}

func (p *PostStore) AddPost(post Post) {
	p.m.Lock()
	defer p.m.Unlock()

	p.data[post.ID] = post
}

func (p *PostStore) GetPost(id string) (Post, bool) {
	p.m.Lock()
	defer p.m.Unlock()

	post, ok := p.data[id]
	return post, ok
}

type FeedStore struct {
	m sync.RWMutex

	data map[int32][]Post
}

func (f *FeedStore) AddPostToUsers(subs []int32, post Post) {
	f.m.Lock()
	defer f.m.Unlock()

	for _, v := range subs {
		cache := f.data[v]

		if cache == nil {
			cache = []Post{}
		}

		cache = append(cache, post)

		f.data[v] = cache
	}
}

func NewFeedStore() FeedStore {
	d := make(map[int32][]Post)
	return FeedStore{
		data: d,
	}
}
