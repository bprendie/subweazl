package localindex

import (
	"context"
	"io/fs"
	"path/filepath"

	"github.com/bprendie/subweazl/internal/localstore"
)

type Prober interface {
	Probe(context.Context, string) (Metadata, error)
}

type Indexer struct {
	Store  *localstore.Store
	Prober Prober
}

type Result struct {
	FolderID string
	Indexed  int
	Skipped  int
}

func (i Indexer) IndexFolder(ctx context.Context, folder string) (Result, error) {
	root, err := filepath.Abs(folder)
	if err != nil {
		return Result{}, err
	}
	folderID := FolderID(root)
	if err := i.Store.UpsertFolder(folderID, map[string]string{"path": root}); err != nil {
		return Result{}, err
	}
	result := Result{FolderID: folderID}
	var present []string
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			result.Skipped++
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			result.Skipped++
			return nil
		}
		meta, err := i.Prober.Probe(ctx, path)
		if err != nil {
			result.Skipped++
			return nil
		}
		meta.Path = path
		meta.FileSize = info.Size()
		meta.ModifiedUnix = info.ModTime().Unix()
		trackID := TrackID(path, info.Size(), info.ModTime().Unix())
		err = i.Store.UpsertTrack(localstore.TrackRecord{
			ID:           trackID,
			FolderID:     folderID,
			FileSize:     info.Size(),
			ModifiedUnix: info.ModTime().Unix(),
			Payload:      meta,
		})
		if err != nil {
			return err
		}
		present = append(present, trackID)
		result.Indexed++
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	if err := i.Store.MarkMissingTracks(folderID, present); err != nil {
		return Result{}, err
	}
	if err := i.Store.CompleteFolderScan(folderID); err != nil {
		return Result{}, err
	}
	return result, nil
}
