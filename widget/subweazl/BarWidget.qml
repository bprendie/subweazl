import QtQuick
import Quickshell
import Quickshell.Io
import qs.Commons
import qs.Ui
import "Model.js" as Model

BarWidget {
  id: root
  moduleName: "io.github.bprendie.subweazl"

  property var status: Model.emptyStatus()
  property bool popupOpen: false
  readonly property bool opened: popupOpen
  readonly property bool popoutSwitchClosing: false
  readonly property bool active: status.running === true
  readonly property bool playing: active && status.state === "playing"
  readonly property bool paused: active && status.state === "paused"
  readonly property string weazlIcon: "󰺢"
  readonly property string stateIcon: playing ? "󰏤" : (paused ? "󰐊" : "󰓛")
  readonly property string tooltipText: Model.trackLabel(status)
  readonly property string home: Quickshell.env("HOME") || ""
  readonly property string defaultSubweazlBin: home + "/.subweazl/bin/subweazl"
  readonly property string subweazlBin: settings && String(settings.binaryPath || "") !== ""
    ? String(settings.binaryPath) : defaultSubweazlBin
  readonly property string configHome: Quickshell.env("XDG_CONFIG_HOME") || (Quickshell.env("HOME") + "/.config")

  function refresh() {
    if (!statusProc.running) statusProc.running = true
  }

  function runRemote(action) {
    if (actionProc.running) return
    actionProc.command = [root.subweazlBin, "remote", action]
    actionProc.running = true
  }

  function launch() {
    if (!launchProc.running) launchProc.running = true
  }

  function close() { popupOpen = false }
  function open() { popupOpen = true }
  function toggle() { popupOpen = !popupOpen }
  function closeForPopoutSwitch() { close() }

  function openSubweazl() {
    close()
    launch()
  }

  implicitWidth: row.implicitWidth + Style.space(14)
  implicitHeight: barSize

  Process {
    id: statusProc
    command: [root.subweazlBin, "remote", "status"]
    stdout: StdioCollector {
      waitForEnd: true
      onStreamFinished: root.status = Model.parseStatus(text)
    }
    onExited: function(exitCode) {
      if (exitCode !== 0) root.status = Model.emptyStatus()
    }
  }

  Process {
    id: actionProc
    onExited: root.refresh()
  }

  Process {
    id: launchProc
    command: [root.configHome + "/omarchy/plugins/io.github.bprendie.subweazl/widget/subweazl/launch.sh", root.subweazlBin]
  }

  Timer {
    interval: 1000
    running: true
    repeat: true
    triggeredOnStart: true
    onTriggered: root.refresh()
  }

  Row {
    id: row
    anchors.centerIn: parent
    spacing: Style.space(6)

    Text {
      anchors.verticalCenter: parent.verticalCenter
      text: root.weazlIcon
      color: root.active ? root.bar.barForeground : Qt.darker(root.bar.barForeground, 1.55)
      font.family: "CaskaydiaMono Nerd Font"
      font.pixelSize: Style.font.icon

      SequentialAnimation on opacity {
        running: root.playing
        loops: Animation.Infinite
        NumberAnimation { to: 0.55; duration: 420; easing.type: Easing.InOutQuad }
        NumberAnimation { to: 1.0; duration: 420; easing.type: Easing.InOutQuad }
      }
    }

    Text {
      anchors.verticalCenter: parent.verticalCenter
      visible: !root.bar.vertical && root.status.title !== ""
      width: Math.min(180, implicitWidth)
      text: root.status.title
      color: root.bar.barForeground
      font.family: root.bar.fontFamily
      font.pixelSize: Style.font.body
      elide: Text.ElideRight
    }
  }

  MouseArea {
    anchors.fill: parent
    hoverEnabled: true
    cursorShape: Qt.PointingHandCursor
    acceptedButtons: Qt.LeftButton | Qt.MiddleButton
    onClicked: function(mouse) {
      if (mouse.button === Qt.MiddleButton && root.active) root.runRemote("next")
      else if (root.active) root.toggle()
      else root.launch()
    }
    onWheel: function(wheel) {
      if (!root.active) return
      root.runRemote(wheel.angleDelta.y > 0 ? "previous" : "next")
    }
    onEntered: if (root.bar) root.bar.showTooltip(root, root.tooltipText)
    onExited: if (root.bar) root.bar.hideTooltip(root)
  }

  PopupCard {
    id: popup
    anchorItem: root
    bar: root.bar
    owner: root
    open: root.popupOpen
    contentWidth: popup.fittedContentWidth(Style.space(320))
    contentHeight: popup.fittedContentHeight(content.implicitHeight)

    Column {
      id: content
      anchors.fill: parent
      spacing: Style.space(10)

      Row {
        width: parent.width
        spacing: Style.space(10)

        BorderSurface {
          width: Style.space(58)
          height: Style.space(58)
          radius: Style.spacing.labelGap
          color: Style.normalFillFor(root.bar.foreground, Color.accent)
          borderSpec: Border.controlSpec("normal", root.bar.foreground, Color.accent)

          Text {
            anchors.centerIn: parent
            text: root.weazlIcon
            color: root.bar.foreground
            font.family: "CaskaydiaMono Nerd Font"
            font.pixelSize: Style.font.displayLarge
          }
        }

        Column {
          width: parent.width - Style.space(68)
          spacing: Style.space(3)

          Text {
            width: parent.width
            text: root.status.title || "Subweazl"
            color: root.bar.foreground
            font.family: root.bar.fontFamily
            font.pixelSize: Style.font.subtitle
            font.bold: true
            elide: Text.ElideRight
          }
          Text {
            width: parent.width
            text: root.status.artist || Model.stateLabel(root.status)
            color: Qt.darker(root.bar.foreground, 1.3)
            font.family: root.bar.fontFamily
            font.pixelSize: Style.font.bodySmall
            elide: Text.ElideRight
          }
          Text {
            width: parent.width
            visible: text !== ""
            text: root.status.album
            color: Qt.darker(root.bar.foreground, 1.6)
            font.family: root.bar.fontFamily
            font.pixelSize: Style.font.caption
            elide: Text.ElideRight
          }
        }
      }

      Row {
        anchors.horizontalCenter: parent.horizontalCenter
        spacing: Style.space(6)

        Button {
          iconText: "󰒮"
          foreground: root.bar.foreground
          enabled: root.active
          opacity: enabled ? 1.0 : 0.4
          onClicked: root.runRemote("previous")
        }
        Button {
          iconText: root.stateIcon
          foreground: root.bar.foreground
          iconSize: Style.font.iconLarge
          enabled: root.active
          opacity: enabled ? 1.0 : 0.4
          onClicked: root.runRemote("toggle")
        }
        Button {
          iconText: "󰒭"
          foreground: root.bar.foreground
          enabled: root.active
          opacity: enabled ? 1.0 : 0.4
          onClicked: root.runRemote("next")
        }
        Button {
          iconText: "󰓛"
          foreground: root.bar.foreground
          enabled: root.active
          opacity: enabled ? 1.0 : 0.4
          onClicked: root.runRemote("stop")
        }
      }

      Button {
        anchors.horizontalCenter: parent.horizontalCenter
        text: root.active ? "Open Subweazl" : "Launch Subweazl"
        foreground: root.bar.foreground
        onClicked: root.openSubweazl()
      }

      Button {
        anchors.horizontalCenter: parent.horizontalCenter
        text: "Mode: " + root.status.playback_mode
        foreground: root.bar.foreground
        enabled: root.active
        opacity: enabled ? 1.0 : 0.4
        onClicked: root.runRemote("mode")
      }

      Button {
        anchors.horizontalCenter: parent.horizontalCenter
        text: "Quit and lock Weazl vault"
        foreground: root.bar.foreground
        enabled: root.active
        opacity: enabled ? 1.0 : 0.4
        onClicked: {
          root.popupOpen = false
          root.runRemote("quit")
        }
      }
    }
  }
}
