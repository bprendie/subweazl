package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bprendie/subweazl/internal/localstore"
	tea "github.com/charmbracelet/bubbletea"
)

func localFolderItems(folders []localstore.FolderSummary, tracks []localstore.TrackSummary, configured []string, open map[string]bool) []item {
	rootSet := map[string]string{}
	for _, folder := range configured {
		rootSet[filepath.Clean(folder)] = "configured - not indexed yet"
	}
	for _, folder := range folders {
		status := "indexed"
		if folder.LastScanCompletedAt != "" {
			status = "last scan " + folder.LastScanCompletedAt
		}
		rootSet[filepath.Clean(folder.Path)] = status
	}
	trackCounts := map[string]int{}
	dirs := map[string]bool{}
	for _, track := range tracks {
		dir := filepath.Clean(filepath.Dir(track.Path))
		for root := range rootSet {
			if !pathWithin(root, dir) {
				continue
			}
			for current := dir; pathWithin(root, current); current = filepath.Dir(current) {
				dirs[current] = true
				trackCounts[current]++
				if current == root || filepath.Dir(current) == current {
					break
				}
			}
		}
	}
	for root := range rootSet {
		dirs[root] = true
	}
	paths := make([]string, 0, len(dirs))
	for path := range dirs {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return folderItemsFromPaths(paths, rootSet, trackCounts, open)
}

func folderItemsFromPaths(paths []string, rootSet map[string]string, trackCounts map[string]int, open map[string]bool) []item {
	items := make([]item, 0, len(paths))
	for _, path := range paths {
		root, ok := nearestRoot(path, rootSet)
		if !ok || !folderNodeVisible(root, path, open) {
			continue
		}
		status := rootSet[path]
		if status == "" {
			status = fmt.Sprintf("%d tracks", trackCounts[path])
		}
		items = append(items, item{
			kind: "local-folder",
			folder: localFolder{
				Path:     path,
				Status:   status,
				Depth:    folderDepth(root, path),
				Expanded: open[path],
				Tracks:   trackCounts[path],
			},
		})
	}
	return items
}

func (m Model) ensureLocalRootsOpen(folders []localstore.FolderSummary) {
	if m.localOpen == nil {
		m.localOpen = map[string]bool{}
	}
	for _, folder := range m.cfg.LocalMusicFolders {
		path := filepath.Clean(folder)
		if _, ok := m.localOpen[path]; !ok {
			m.localOpen[path] = true
		}
	}
	for _, folder := range folders {
		path := filepath.Clean(folder.Path)
		if _, ok := m.localOpen[path]; !ok {
			m.localOpen[path] = true
		}
	}
}

func (m Model) toggleLocalFolder(folder localFolder) (Model, tea.Cmd) {
	if m.localOpen == nil {
		m.localOpen = map[string]bool{}
	}
	m.localOpen[folder.Path] = !m.localOpen[folder.Path]
	next, cmd := m.renderUnlockedLocal()
	next.status = folder.Path
	return next, cmd
}

func folderVisible(trackPath string, open map[string]bool) bool {
	dir := filepath.Clean(filepath.Dir(trackPath))
	for {
		if expanded, ok := open[dir]; ok && !expanded {
			return false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return true
		}
		dir = parent
	}
}

func folderNodeVisible(root, path string, open map[string]bool) bool {
	if path == root {
		return true
	}
	parent := filepath.Dir(path)
	for parent != root && parent != filepath.Dir(parent) {
		if !open[parent] {
			return false
		}
		parent = filepath.Dir(parent)
	}
	return open[root]
}

func pathWithin(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	return path == root || strings.HasPrefix(path, root+string(filepath.Separator))
}

func nearestRoot(path string, roots map[string]string) (string, bool) {
	best := ""
	for root := range roots {
		if pathWithin(root, path) && len(root) > len(best) {
			best = root
		}
	}
	return best, best != ""
}

func folderDepth(root, path string) int {
	if root == path {
		return 0
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}
