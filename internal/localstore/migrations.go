package localstore

func (s *Store) Migrate() error {
	stmts := []string{
		`create table if not exists vault (
			id integer primary key check (id = 1),
			password_hash text not null,
			created_at datetime not null default current_timestamp
		)`,
		`create table if not exists folders (
			id text primary key,
			payload text not null,
			last_scan_started_at datetime,
			last_scan_completed_at datetime,
			created_at datetime not null default current_timestamp,
			updated_at datetime not null default current_timestamp
		)`,
		`create table if not exists tracks (
			id text primary key,
			folder_id text references folders(id) on delete set null,
			payload text not null,
			file_hash text,
			file_size integer not null default 0,
			modified_unix integer not null default 0,
			missing integer not null default 0,
			created_at datetime not null default current_timestamp,
			updated_at datetime not null default current_timestamp
		)`,
		`create table if not exists albums (
			id text primary key,
			payload text not null,
			created_at datetime not null default current_timestamp,
			updated_at datetime not null default current_timestamp
		)`,
		`create table if not exists artists (
			id text primary key,
			payload text not null,
			created_at datetime not null default current_timestamp,
			updated_at datetime not null default current_timestamp
		)`,
		`create table if not exists local_playlists (
			id text primary key,
			kind text not null default 'manual',
			payload text not null,
			created_at datetime not null default current_timestamp,
			updated_at datetime not null default current_timestamp
		)`,
		`create table if not exists local_playlist_tracks (
			playlist_id text not null references local_playlists(id) on delete cascade,
			position integer not null,
			source text not null check (source in ('local', 'subsonic')),
			track_id text not null,
			created_at datetime not null default current_timestamp,
			primary key (playlist_id, position)
		)`,
		`create table if not exists play_history (
			id integer primary key autoincrement,
			source text not null check (source in ('local', 'subsonic')),
			track_id text not null,
			payload text not null,
			played_at datetime not null default current_timestamp
		)`,
		`create table if not exists ratings (
			source text not null check (source in ('local', 'subsonic')),
			track_id text not null,
			payload text not null,
			updated_at datetime not null default current_timestamp,
			primary key (source, track_id)
		)`,
		`create table if not exists station_recipes (
			id text primary key,
			payload text not null,
			created_at datetime not null default current_timestamp,
			updated_at datetime not null default current_timestamp
		)`,
		`create table if not exists recommendation_runs (
			id text primary key,
			provider text not null,
			model text not null,
			payload text not null,
			created_at datetime not null default current_timestamp
		)`,
		`create index if not exists idx_tracks_folder on tracks(folder_id)`,
		`create index if not exists idx_tracks_file_hash on tracks(file_hash)`,
		`create index if not exists idx_tracks_missing on tracks(missing)`,
		`create index if not exists idx_local_playlist_tracks_playlist on local_playlist_tracks(playlist_id, position)`,
		`create index if not exists idx_local_playlist_tracks_track on local_playlist_tracks(source, track_id)`,
		`create index if not exists idx_play_history_recent on play_history(played_at desc)`,
		`create index if not exists idx_play_history_track on play_history(source, track_id)`,
		`create index if not exists idx_local_playlists_updated on local_playlists(updated_at desc)`,
		`create index if not exists idx_recommendation_runs_created on recommendation_runs(created_at desc)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
