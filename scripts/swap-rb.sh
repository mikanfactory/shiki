#!/bin/bash
set -euo pipefail

session=$(tmux display-message -p '#{session_name}')
tmux swap-pane -d -s "${session}:main-window.2" -t "${session}:background-window.2"
tmux swap-pane -d -s "${session}:background-window.2" -t "${session}:background-window.3"
