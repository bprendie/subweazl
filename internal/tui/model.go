package tui

import (
	"fmt"
	"image"
	"time"

	"github.com/bprendie/subweazl/internal/audio"
	"github.com/bprendie/subweazl/internal/config"
	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/bprendie/subweazl/internal/player"
	"github.com/bprendie/subweazl/internal/playqueue"
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
	modeHome mode = iota
	modeNewest
	modeRandomAlbums
	modePlaylists
	modeTracks
	modeSearch
	modeQueue
	modePrivatePlaylists
	modeStation
	modeLastPlayed
	modeSetup
	modeVault
)

const searchPrompt = "stream > "

type Model struct {
	cfg             config.Config
	styles          styles
	client          subsonic.Client
	player          *player.Player
	meter           *audio.Meter
	list            list.Model
	input           textinput.Model
	setup           []textinput.Model
	setupFocus      int
	vaultInput      textinput.Model
	vaultStore      *localstore.Store
	vaultStage      string
	vaultPass       string
	spinner         spinner.Model
	mode            mode
	nav             []navSnapshot
	appState        state.State
	status          string
	err             string
	searching       bool
	playing         *subsonic.Track
	playSource      string
	queue           playqueue.Queue
	cacheStatus     localstore.CacheStatus
	paused          bool
	trackTitle      string
	titlePoll       time.Time
	coverID         string
	coverArt        image.Image
	coverErr        string
	coverCache      map[string]image.Image
	energy          audio.Sample
	renaming        *subsonic.Playlist
	savingQueue     bool
	privateRenaming string
	station         *subsonic.Playlist
	width           int
	height          int
	visualizer      Visualizer
}

type item struct {
	kind            string
	title           string
	desc            string
	track           subsonic.Track
	album           subsonic.Album
	playlist        subsonic.Playlist
	privatePlaylist localstore.PrivatePlaylist
	action          string
	queueIndex      int
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
	case "private_playlist":
		return i.privatePlaylist.Name
	case "empty", "home", "queue":
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
	case "private_playlist":
		return fmt.Sprintf("private vault playlist  %d tracks", len(i.privatePlaylist.Tracks))
	case "empty", "home", "queue":
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

func New(cfg config.Config) Model {
	input := textinput.New()
	input.Placeholder = "song, artist, or album"
	input.Prompt = searchPrompt
	input.CharLimit = 240
	input.Width = 42
	vaultInput := textinput.New()
	vaultInput.EchoMode = textinput.EchoPassword
	vaultInput.CharLimit = 240
	vaultInput.Width = 42
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
		vaultInput: vaultInput,
		spinner:    newSpinner(),
		mode:       modeHome,
		status:     "ready",
		appState:   appState,
		coverCache: map[string]image.Image{},
		queue:      playqueue.New(),
		visualizer: NewVisualizer(harmonica.FPS(30)),
	}
	if stateErr != nil {
		m.err = stateErr.Error()
	}
	m.setup = newSetupInputs(cfg)
	if !cfg.Ready() {
		m.mode = modeSetup
		m.status = "connect a Subsonic server"
		m.refreshTitle()
		return m
	}
	if err := m.prepareVault(); err != nil {
		m.err = err.Error()
	}
	if m.mode != modeVault {
		m.restoreQueueSnapshot()
		m.refreshCacheStatus()
		m.showHome()
	}
	m.refreshTitle()
	return m
}

func (m Model) Init() tea.Cmd {
	if !m.cfg.Ready() || m.mode == modeVault {
		return tick()
	}
	return tick()
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

func (m *Model) refreshTitle() {
	switch m.mode {
	case modeHome:
		m.list.Title = "home"
	case modePlaylists:
		m.list.Title = "playlists"
	case modeRandomAlbums:
		m.list.Title = "random albums"
	case modeTracks:
		m.list.Title = "tracks"
	case modeSearch:
		m.list.Title = "search results"
	case modeQueue:
		m.list.Title = "queue"
	case modePrivatePlaylists:
		m.list.Title = "private playlists"
	case modeStation:
		m.list.Title = "station"
	case modeLastPlayed:
		m.list.Title = "last played"
	case modeSetup:
		m.list.Title = "setup"
	case modeVault:
		m.list.Title = "private vault"
	default:
		m.list.Title = "newest albums"
	}
}
