package config

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"user_token", Config{SystemURL: "http://s", AuthMode: AuthUserToken}, false},
		{"none", Config{SystemURL: "http://s", AuthMode: AuthNone}, false},
		{"service_token с ключом", Config{SystemURL: "http://s", AuthMode: AuthServiceToken, ServiceToken: "k"}, false},
		{"service_token без ключа", Config{SystemURL: "http://s", AuthMode: AuthServiceToken}, true},
		{"нет SYSTEM_URL", Config{AuthMode: AuthNone}, true},
		{"неизвестный режим", Config{SystemURL: "http://s", AuthMode: "oauth"}, true},
		{"ключ платформы с user_token и ключом", Config{SystemURL: "http://s", AuthMode: AuthUserToken, ServiceToken: "k", RequirePlatformKey: true}, false},
		{"ключ платформы без SERVICE_TOKEN", Config{SystemURL: "http://s", AuthMode: AuthUserToken, RequirePlatformKey: true}, true},
		{"ключ платформы с service_token", Config{SystemURL: "http://s", AuthMode: AuthServiceToken, ServiceToken: "k", RequirePlatformKey: true}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() = %v, ожидали ошибку: %v", err, tt.wantErr)
			}
		})
	}
}

func TestNoneWithSystemTokenRejected(t *testing.T) {
	c := &Config{SystemURL: "http://sys", AuthMode: AuthNone, SystemToken: "server-key"}
	if c.Validate() == nil {
		t.Fatal("AUTH_MODE=none с SYSTEM_TOKEN должен отклоняться")
	}
	c.SystemToken = ""
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}
