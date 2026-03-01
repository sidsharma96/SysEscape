package roomctl

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var epoch = time.Unix(0, 0)

type BuildResult struct {
	BundlePath   string
	ManifestPath string
	Manifest     Manifest
}

type bundleEntry struct {
	abs   string
	rel   string
	isDir bool
}

func BuildRoom(roomDir string, version int) (*BuildResult, error) {
	if version < 1 {
		return nil, fmt.Errorf("version must be a positive integer")
	}
	if err := ValidateRoomDir(roomDir); err != nil {
		return nil, err
	}

	var meta Metadata
	if err := parseYAML(filepath.Join(roomDir, "metadata.yaml"), &meta); err != nil {
		return nil, fmt.Errorf("metadata.yaml: %w", err)
	}

	entries, files, err := collectBundleEntries(roomDir)
	if err != nil {
		return nil, err
	}
	if meta.Engine == "B" {
		if err := checkEngineBLeak(files); err != nil {
			return nil, err
		}
	}

	tarBytes, err := createDeterministicTar(entries)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(tarBytes)
	hashHex := hex.EncodeToString(hash[:])

	outDir := filepath.Join(".build", meta.Slug)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	bundlePath := filepath.Join(outDir, "bundle.tar")
	if err := os.WriteFile(bundlePath, tarBytes, 0o644); err != nil {
		return nil, err
	}

	manifest := Manifest{
		SchemaVersion:    1,
		Slug:             meta.Slug,
		Engine:           meta.Engine,
		Version:          version,
		BundleHashSha256: hashHex,
		Files:            files,
		BuiltAt:          time.Now().UTC().Format(time.RFC3339),
	}
	manifestPath := filepath.Join(outDir, "manifest.json")
	if err := writeManifest(manifestPath, manifest); err != nil {
		return nil, err
	}

	return &BuildResult{BundlePath: bundlePath, ManifestPath: manifestPath, Manifest: manifest}, nil
}

func collectBundleEntries(root string) ([]bundleEntry, []string, error) {
	entries := []bundleEntry{}
	files := []string{}

	var walk func(rel string) error
	walk = func(rel string) error {
		dir := filepath.Join(root, rel)
		ds, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, d := range ds {
			nextRel := filepath.Join(rel, d.Name())
			abs := filepath.Join(root, nextRel)
			if d.IsDir() {
				entries = append(entries, bundleEntry{abs: abs, rel: filepath.ToSlash(nextRel), isDir: true})
				if err := walk(nextRel); err != nil {
					return err
				}
				continue
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("unsupported file type: %s", abs)
			}
			relSlash := filepath.ToSlash(nextRel)
			entries = append(entries, bundleEntry{abs: abs, rel: relSlash})
			files = append(files, relSlash)
		}
		return nil
	}

	if err := walk(""); err != nil {
		return nil, nil, err
	}
	return entries, files, nil
}

func createDeterministicTar(entries []bundleEntry) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	tw := tar.NewWriter(buf)
	for _, e := range entries {
		if e.isDir {
			h := &tar.Header{
				Name:       e.rel + "/",
				Typeflag:   tar.TypeDir,
				Mode:       0o755,
				Uid:        0,
				Gid:        0,
				Uname:      "",
				Gname:      "",
				ModTime:    epoch,
				AccessTime: epoch,
				ChangeTime: epoch,
			}
			if err := tw.WriteHeader(h); err != nil {
				_ = tw.Close()
				return nil, err
			}
			continue
		}
		b, err := os.ReadFile(e.abs)
		if err != nil {
			_ = tw.Close()
			return nil, err
		}
		h := &tar.Header{
			Name:       e.rel,
			Typeflag:   tar.TypeReg,
			Mode:       0o644,
			Size:       int64(len(b)),
			Uid:        0,
			Gid:        0,
			Uname:      "",
			Gname:      "",
			ModTime:    epoch,
			AccessTime: epoch,
			ChangeTime: epoch,
		}
		if err := tw.WriteHeader(h); err != nil {
			_ = tw.Close()
			return nil, err
		}
		if _, err := tw.Write(b); err != nil {
			_ = tw.Close()
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func checkEngineBLeak(files []string) error {
	for _, f := range files {
		s := filepath.ToSlash(f)
		if strings.HasPrefix(s, "engineB/workspace/judge/hidden_tests/") {
			return fmt.Errorf("engine B leak check failed: hidden tests found in workspace: %s", s)
		}
	}
	return nil
}
