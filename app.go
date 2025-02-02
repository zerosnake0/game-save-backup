package main

import (
	"bytes"
	"context"
	stderr "errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

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
	subDir := filepath.Join(root, name)
	return os.RemoveAll(subDir)
}

func (a *App) Open(name string) error {
	subDir := filepath.Join(root, name)
	cmd := "open"
	if runtime.GOOS == "windows" {
		cmd = "explorer"
	}
	return exec.Command(cmd, subDir).Start()
}

func (a *App) Files(name string) ([]string, error) {
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

func (a *App) saveFiles(name string, files map[string]bool) error {
	arr := make([]string, 0, len(files))
	for file := range files {
		arr = append(arr, file)
	}
	sort.Strings(arr)
	configFile := filepath.Join(root, name, "config")
	return os.WriteFile(configFile, []byte(strings.Join(arr, "\n")), 0644)
}

func (a *App) AddFiles(name string, files []string) error {
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

func (a *App) Backup(name string) (map[string]bool, error) {
	files, err := a.Files(name)
	if err != nil {
		return nil, err
	}

	realFiles := map[string]bool{}
	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			return nil, err
		}
		if stat.IsDir() {
			fmt.Println("dir", file)
			err = filepath.WalkDir(file, func(path string, d fs.DirEntry, err error) error {
				fmt.Println("dir2", path)
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
				return nil, err
			}
		} else {
			fmt.Println("file", file)
			realFiles[file] = true
		}
	}
	return realFiles, nil
	//buf := bytes.NewBuffer(nil)
	//zw := zip.NewWriter(buf)
	//err = zw.Close()
	//if err != nil {
	//	return nil, err
	//}
	//for file := range realFiles {
	//	w, err := zw.CreateHeader(&zip.FileHeader{
	//		Name: file,
	//	})
	//	if err != nil {
	//		return nil, err
	//	}
	//}
}
