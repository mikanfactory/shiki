#!/bin/bash
set -euo pipefail

session=$(tmux display-message -p '#{session_name}')
tmux swap-pane -d -s "${session}:main-window.0" -t "${session}:background-window.0"
tmux swap-pane -d -s "${session}:background-window.0" -t "${session}:background-window.1"
