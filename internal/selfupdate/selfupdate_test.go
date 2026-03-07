package selfupdate

import "testing"

func TestSelectAssetFallsBackToFirstAsset(t *testing.T) {
	t.Parallel()

	rel := &githubRelease{
		Assets: []githubAsset{
			{
				Name:               "forge-linux-arm64",
				BrowserDownloadURL: "https://example.com/forge-linux-arm64",
			},
		},
	}

	url, name, err := selectAsset(rel)
	if err == nil {
		t.Fatal("expected fallback warning error, got nil")
	}
	if url != "https://example.com/forge-linux-arm64" {
		t.Fatalf("url = %q", url)
	}
	if name != "forge-linux-arm64" {
		t.Fatalf("name = %q", name)
	}
}
