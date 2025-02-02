package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	stderr "errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	wailsRT "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

var (
	root string
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	root = filepath.Join(home, "game_saves")
	err = os.Mkdir(root, 0755)
	if err != nil {
		if !os.IsExist(err) {
			panic(err)
		}
	}
}

func (a *App) Root() string {
	return root
}

// Greet returns a greeting for the given name
func (a *App) List() (arr []string, _ error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			arr = append(arr, entry.Name())
		}
	}
	return arr, nil
}

func (a *App) Add(name string) error {
	if name == "" {
		return stderr.New("empty name")
	}
	subDir := filepath.Join(root, name)
	return os.Mkdir(subDir, 0755)
}

func (a *App) Remove(name string) error {
	if name == "" {
		return stderr.New("empty name")
	}
	subDir := filepath.Join(root, name)
	return os.RemoveAll(subDir)
}

func (a *App) Open(name string) error {
	// name可以空
	subDir := filepath.Join(root, name)
	cmd := "open"
	if runtime.GOOS == "windows" {
		cmd = "explorer"
	}
	return exec.Command(cmd, subDir).Start()
}

func (a *App) Files(name string) ([]string, error) {
	if name == "" {
		return nil, stderr.New("empty name")
	}

	configFile := filepath.Join(root, name, "config")
	b, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	arr := bytes.Split(b, []byte{'\n'})
	ret := make([]string, 0, len(arr))
	for _, file := range arr {
		s := bytes.TrimSpace(file)
		if len(s) > 0 {
			ret = append(ret, string(s))
		}
	}
	return ret, nil
}

func (a *App) ChooseFiles() ([]string, error) {
	return wailsRT.OpenMultipleFilesDialog(a.ctx, wailsRT.OpenDialogOptions{})
}

func (a *App) ChooseDir() (string, error) {
	return wailsRT.OpenDirectoryDialog(a.ctx, wailsRT.OpenDialogOptions{})
}

func map2slice(m map[string]bool) []string {
	ret := make([]string, 0, len(m))
	for k := range m {
		ret = append(ret, k)
	}
	sort.Strings(ret)
	return ret
}

func (a *App) saveFiles(name string, files map[string]bool) error {
	arr := map2slice(files)
	configFile := filepath.Join(root, name, "config")
	return os.WriteFile(configFile, []byte(strings.Join(arr, "\n")), 0644)
}

func (a *App) AddFiles(name string, files []string) error {
	if name == "" {
		return stderr.New("empty name")
	}
	if len(files) == 0 {
		return nil
	}
	current, err := a.Files(name)
	if err != nil {
		return err
	}
	m := map[string]bool{}
	for _, file := range current {
		m[file] = true
	}
	for _, file := range files {
		m[file] = true
	}
	return a.saveFiles(name, m)
}

func (a *App) RemoveFile(name, file string) error {
	if name == "" {
		return stderr.New("empty name")
	}
	if file == "" {
		return stderr.New("empty filename")
	}
	current, err := a.Files(name)
	if err != nil {
		return err
	}
	m := map[string]bool{}
	for _, v := range current {
		m[v] = true
	}
	delete(m, file)
	return a.saveFiles(name, m)
}

func (a *App) Backups(name string) (arr []string, _ error) {
	if name == "" {
		return nil, stderr.New("empty name")
	}

	subDir := filepath.Join(root, name)
	entries, err := os.ReadDir(subDir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		entryName := entry.Name()
		if strings.HasSuffix(entryName, ".zip") {
			arr = append(arr, entryName)
		}
	}
	return arr, nil
}

func (a *App) Backup(name string) (res any, err error) {
	return a.backup(name, false)
}

func (a *App) backup(name string, auto bool) (res any, err error) {
	if name == "" {
		return res, stderr.New("empty name")
	}

	files, err := a.Files(name)
	if err != nil {
		return res, err
	}

	realFiles := map[string]bool{}
	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return res, err
		}
		if stat.IsDir() {
			err = filepath.WalkDir(file, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				realFiles[path] = true
				return nil
			})
			if err != nil {
				return res, err
			}
		} else {
			realFiles[file] = true
		}
	}

	sorted := map2slice(realFiles)

	hash := md5.New()
	buf := bytes.NewBuffer(nil)
	zw := zip.NewWriter(buf)
	for _, file := range sorted {
		nameInZip := filepath.ToSlash(file)
		if len(nameInZip) == 0 {
			return res, stderr.New("empty file name in zip")
		}
		if nameInZip[0] != '/' {
			nameInZip = "/" + strings.ReplaceAll(nameInZip, ":", "")
		}
		w, err := zw.CreateHeader(&zip.FileHeader{
			Name:    nameInZip,
			Comment: file,
			Method:  zip.Deflate,
		})
		if err != nil {
			return res, err
		}
		b, err := os.ReadFile(file)
		if err != nil {
			return res, err
		}
		_, err = w.Write(b)
		if err != nil {
			return res, err
		}
		_, err = hash.Write(b)
		if err != nil {
			return res, err
		}
	}
	err = zw.Close()
	if err != nil {
		return res, err
	}

	md5 := hex.EncodeToString(hash.Sum(nil))
	suffix := ""
	if auto {
		suffix = "_auto"
	}
	zipFileName := fmt.Sprintf("%s_%s_%s%s.zip", name, time.Now().Format("20060102_150405"), md5, suffix)
	err = os.WriteFile(filepath.Join(root, name, zipFileName), buf.Bytes(), 0644)
	return nil, err
}

func (a *App) RemoveOne(name, file string) error {
	if name == "" {
		return stderr.New("empty name")
	}
	if file == "" {
		return stderr.New("empty filename")
	}
	return os.Remove(filepath.Join(root, name, file))
}

func (a *App) Restore(name, file string) error {
	if name == "" {
		return stderr.New("empty name")
	}
	if file == "" {
		return stderr.New("empty filename")
	}
	if _, err := a.backup(name, true); err != nil {
		return err
	}
	subPath := filepath.Join(root, name, file)
	zr, err := zip.OpenReader(subPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		targetPath := f.Comment
		if targetPath == "" {
			return stderr.New("empty target path")
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		b, err := io.ReadAll(rc)
		if err != nil {
			return err
		}
		err = rc.Close()
		if err != nil {
			return err
		}
		err = os.WriteFile(targetPath, b, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}
