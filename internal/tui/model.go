package tui

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"
	"time"

	"github.com/bprendie/subweazl/internal/audio"
	"github.com/bprendie/subweazl/internal/config"
	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/player"
	"github.com/bprendie/subweazl/internal/state"
	"github.com/bprendie/subweazl/internal/subsonic"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/lipgloss"
)

type mode int

const (
	modeSources mode = iota
	modeNewest
	modeRandomAlbums
	modePlaylists
	modeTracks
	modeSearch
	modeStation
	modeLastPlayed
	modeLocal
	modeSetup
)

const searchPrompt = "stream > "

type Model struct {
	cfg        config.Config
	styles     styles
	client     subsonic.Client
	player     *player.Player
	meter      *audio.Meter
	list       list.Model
	input      textinput.Model
	setup      []textinput.Model
	setupFocus int
	spinner    spinner.Model
	mode       mode
	nav        []navSnapshot
	appState   state.State
	status     string
	err        string
	searching  bool
	playing    *subsonic.Track
	localPlay  *localTrack
	playSource string
	paused     bool
	trackTitle string
	titlePoll  time.Time
	coverID    string
	coverArt   image.Image
	coverErr   string
	coverCache map[string]image.Image
	energy     audio.Sample
	renaming   *subsonic.Playlist
	station    *subsonic.Playlist
	localStore *localstore.Store
	localVault string
	localPass  string
	localOpen  map[string]bool
	width      int
	height     int
	visualizer Visualizer
}

type item struct {
	kind     string
	title    string
	desc     string
	track    subsonic.Track
	album    subsonic.Album
	playlist subsonic.Playlist
	folder   localFolder
	local    localTrack
	action   string
}

type localFolder struct {
	ID       string
	Path     string
	Status   string
	Depth    int
	Expanded bool
	Tracks   int
}

type localTrack struct {
	ID      string
	Title   string
	Artist  string
	Album   string
	Path    string
	Dir     string
	Missing bool
}

type navSnapshot struct {
	mode   mode
	items  []list.Item
	status string
	cursor int
}

func (i item) Title() string {
	switch i.kind {
	case "album":
		return i.album.Name
	case "playlist":
		return i.playlist.Name
	case "local-folder":
		return i.folder.Path
	case "local-song":
		if i.local.Title != "" {
			return i.local.Title
		}
		name := filepath.Base(i.local.Path)
		ext := filepath.Ext(name)
		return strings.TrimSuffix(name, ext)
	case "source", "empty":
		return i.title
	case "action":
		return i.title
	default:
		return i.track.Title
	}
}

func (i item) Description() string {
	switch i.kind {
	case "album":
		return fmt.Sprintf("%s  %d", i.album.Artist, i.album.Year)
	case "playlist":
		return fmt.Sprintf("%d tracks", i.playlist.Count)
	case "local-folder":
		return i.folder.Status
	case "local-song":
		return localTrackDescription(i.local)
	case "source", "empty":
		return i.desc
	case "action":
		return i.desc
	default:
		return fmt.Sprintf("%s  %s", i.track.Artist, i.track.Album)
	}
}

func (i item) FilterValue() string { return i.Title() + " " + i.Description() }

type loadedMsg struct {
	items  []list.Item
	status string
	mode   mode
}

type stationMsg struct {
	playlist subsonic.Playlist
	tracks   []subsonic.Track
}

type errMsg struct{ err error }
type renamedMsg struct {
	id   string
	name string
}
type titleMsg struct{ title string }
type tickMsg time.Time
type setupSavedMsg struct {
	cfg    config.Config
	status string
}
type coverArtMsg struct {
	id  string
	img image.Image
	err error
}

type localIndexedMsg struct {
	folders int
	indexed int
	skipped int
}

func New(cfg config.Config) Model {
	input := textinput.New()
	input.Placeholder = "song, artist, or album"
	input.Prompt = searchPrompt
	input.CharLimit = 240
	input.Width = 42
	l := list.New(nil, delegate{styles: newStyles()}, 80, 20)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	appState, stateErr := state.Load()
	m := Model{
		cfg:        cfg,
		styles:     newStyles(),
		client:     subsonic.New(cfg.Server, cfg.Username, cfg.Password),
		player:     player.New(),
		list:       l,
		input:      input,
		spinner:    newSpinner(),
		mode:       modeNewest,
		status:     "ready",
		appState:   appState,
		coverCache: map[string]image.Image{},
		localOpen:  map[string]bool{},
		visualizer: NewVisualizer(harmonica.FPS(30)),
	}
	if stateErr != nil {
		m.err = stateErr.Error()
	}
	m.setup = newSetupInputs(cfg)
	if !cfg.Ready() {
		m.mode = modeSetup
		m.status = "connect a Subsonic server or add local music folders"
	}
	m.restoreLastPlayed()
	m.refreshTitle()
	return m
}

func (m Model) Init() tea.Cmd {
	if !m.cfg.Ready() {
		return tick()
	}
	if m.hasRestoredLastPlayed() {
		return tea.Batch(tick(), m.loadCoverArt(m.coverID))
	}
	return tea.Batch(tick(), m.loadNewest())
}

func newSetupInputs(cfg config.Config) []textinput.Model {
	fields := []struct {
		label       string
		value       string
		placeholder string
		width       int
	}{
		{"server", cfg.Server, "https://your-navidrome.example", 52},
		{"username", cfg.Username, "subsonic username", 32},
		{"password", cfg.Password, "subsonic password", 32},
		{"folders", joinFolders(cfg.LocalMusicFolders), "/music, ~/Music", 52},
	}
	inputs := make([]textinput.Model, 0, len(fields))
	for i, field := range fields {
		in := textinput.New()
		in.Prompt = field.label + " > "
		in.Placeholder = field.placeholder
		in.SetValue(field.value)
		in.Width = field.width
		if field.label == "password" {
			in.EchoMode = textinput.EchoPassword
		}
		if i == 0 {
			in.Focus()
		}
		inputs = append(inputs, in)
	}
	return inputs
}

func newSpinner() spinner.Model {
	frames := []string{
		lipgloss.NewStyle().Foreground(crushGold).Render("●"),
		lipgloss.NewStyle().Foreground(crushPink).Render("●"),
		lipgloss.NewStyle().Foreground(crushPurple).Render("●"),
		lipgloss.NewStyle().Foreground(crushMint).Render("●"),
	}
	return spinner.New(spinner.WithSpinner(spinner.Spinner{Frames: frames, FPS: time.Second / 8}))
}

func (m *Model) restoreLastPlayed() {
	if m.mode == modeSetup {
		return
	}
	if m.appState.LastPlayed == nil {
		return
	}
	track := m.appState.LastPlayed.Track()
	if track.ID == "" {
		return
	}
	m.mode = modeLastPlayed
	m.coverID = coverArtID(track)
	m.list.SetItems([]list.Item{item{kind: "song", track: track}})
	m.status = "last played: " + track.Title
}

func (m Model) hasRestoredLastPlayed() bool {
	return m.appState.LastPlayed != nil && len(m.list.Items()) > 0
}

func (m *Model) refreshTitle() {
	switch m.mode {
	case modePlaylists:
		m.list.Title = "playlists"
	case modeRandomAlbums:
		m.list.Title = "random albums"
	case modeTracks:
		m.list.Title = "tracks"
	case modeSearch:
		m.list.Title = "search results"
	case modeStation:
		m.list.Title = "station"
	case modeLastPlayed:
		m.list.Title = "last played"
	case modeLocal:
		m.list.Title = "local library"
	case modeSetup:
		m.list.Title = "setup"
	case modeSources:
		m.list.Title = "sources"
	default:
		m.list.Title = "newest albums"
	}
}

func (m *Model) showSources() {
	m.mode = modeSources
	m.clearNav()
	m.refreshTitle()
	m.list.SetItems([]list.Item{
		item{kind: "source", title: "Subsonic", desc: m.serverLabel()},
		item{kind: "source", title: "Local", desc: m.localLabel()},
	})
	m.status = "choose a source"
	m.err = ""
	m.searching = false
	m.input.Blur()
}
