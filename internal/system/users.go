package system

import "context"

// User — профиль сотрудника в системе. Только то, что пригодится модели и журналу.
type User struct {
	ID         string `json:"id" jsonschema:"идентификатор сотрудника в системе"`
	Name       string `json:"name" jsonschema:"ФИО"`
	Email      string `json:"email,omitempty"`
	Department string `json:"department,omitempty" jsonschema:"подразделение"`
}

// Me возвращает владельца токена: GET /me. Ошибка ErrUnauthorized — токен не принят.
//
// ЗАГЛУШКА: замените путь и поля на настоящий вызов «кто я» вашей системы.
func (c *Client) Me(ctx context.Context, token string) (*User, error) {
	var u User
	if err := c.get(ctx, token, "/me", nil, &u); err != nil {
		return nil, err
	}
	return &u, nil
}
