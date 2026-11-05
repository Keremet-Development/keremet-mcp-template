package system

import (
	"context"
	"net/url"
	"strconv"
)

// Item — одна запись системы в том виде, в каком она нужна модели: коротко, без служебных полей.
type Item struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty" jsonschema:"дата изменения, ГГГГ-ММ-ДД"`
}

// SearchItems ищет записи, видимые владельцу токена: GET /items?q=...&limit=...
//
// ЗАГЛУШКА: замените путь, параметры и поля на настоящий поиск вашей системы.
func (c *Client) SearchItems(ctx context.Context, token, query string, limit int) ([]Item, error) {
	var resp struct {
		Items []Item `json:"items"`
	}

	q := url.Values{"q": {query}, "limit": {strconv.Itoa(limit)}}
	if err := c.get(ctx, token, "/items", q, &resp); err != nil {
		return nil, err
	}

	return resp.Items, nil
}
