package database

import "testing"

func TestParseForgeDBDSN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantDriver string
		wantDSN    string
		wantErr    bool
	}{
		{
			name:       "sqlite url",
			input:      "sqlite://database/database.db",
			wantDriver: "sqlite",
			wantDSN:    "database/database.db",
		},
		{
			name:       "sqlite bare path",
			input:      "database/database.db",
			wantDriver: "sqlite",
			wantDSN:    "database/database.db",
		},
		{
			name:       "postgres url",
			input:      "postgres://user:pass@localhost:5432/app?sslmode=disable",
			wantDriver: "postgres",
			wantDSN:    "postgres://user:pass@localhost:5432/app?sslmode=disable",
		},
		{
			name:       "mysql url",
			input:      "mysql://user:pass@localhost:3306/app",
			wantDriver: "mysql",
			wantDSN:    "user:pass@tcp(localhost:3306)/app?charset=utf8mb4&loc=Local&parseTime=True",
		},
		{
			name:    "empty",
			input:   "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotDriver, gotDSN, err := parseForgeDBDSN(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotDriver != tt.wantDriver {
				t.Fatalf("driver = %q, want %q", gotDriver, tt.wantDriver)
			}
			if gotDSN != tt.wantDSN {
				t.Fatalf("dsn = %q, want %q", gotDSN, tt.wantDSN)
			}
		})
	}
}
