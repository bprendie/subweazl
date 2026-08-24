.pragma library

function emptyStatus() {
  return {
    running: false,
    state: "stopped",
    title: "",
    artist: "",
    album: "",
    duration: 0,
    playback_mode: "off",
    updated_at: ""
  }
}

function parseStatus(text) {
  try {
    var value = JSON.parse(String(text || ""))
    if (!value || typeof value !== "object") return emptyStatus()
    return {
      running: value.running === true,
      state: String(value.state || "stopped"),
      title: String(value.title || ""),
      artist: String(value.artist || ""),
      album: String(value.album || ""),
      duration: Number(value.duration || 0),
      playback_mode: String(value.playback_mode || "off"),
      updated_at: String(value.updated_at || "")
    }
  } catch (error) {
    return emptyStatus()
  }
}

function stateLabel(status) {
  if (!status.running) return "Subweazl is not running"
  if (status.state === "paused") return "Paused"
  if (status.state === "playing") return "Playing"
  return "Idle"
}

function trackLabel(status) {
  if (!status.title) return stateLabel(status)
  return status.artist ? status.title + " — " + status.artist : status.title
}
