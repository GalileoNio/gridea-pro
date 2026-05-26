package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyPostAssetDirsCopiesTyporaAssetsToPostOutput(t *testing.T) {
	appDir := t.TempDir()
	buildDir := t.TempDir()

	srcDir := filepath.Join(appDir, "posts", "hello-world.assets")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("create post asset dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "image.png"), []byte("asset"), 0644); err != nil {
		t.Fatalf("write post asset: %v", err)
	}

	manager := NewAssetManager(appDir, nil)
	manager.SetManifest(NewRenderManifest(buildDir))

	if err := manager.CopyPostAssetDirs(buildDir, ""); err != nil {
		t.Fatalf("CopyPostAssetDirs failed: %v", err)
	}

	dest := filepath.Join(buildDir, DefaultPostPath, "hello-world", "hello-world.assets", "image.png")
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("copied post asset missing: %v", err)
	}
	if string(data) != "asset" {
		t.Fatalf("copied post asset content = %q, want %q", string(data), "asset")
	}

	files := manager.manifest.Files()
	if _, ok := files["post/hello-world/hello-world.assets/image.png"]; !ok {
		t.Fatalf("manifest did not track copied post asset: %#v", files)
	}
}
