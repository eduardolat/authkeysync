# This file will be copied to the devcontainer's .bashrc

alias t='task'
alias td='task dev'
alias tb='task build'
alias tt='task test'
alias tl='task lint'
alias tf='task format'
alias tr='task run'
alias ll='ls -alF'
alias la='ls -A'
alias l='ls -CF'
alias ..='cd ..'
alias c='clear'

umask 000
chmod -R 777 /workspaces

git config --global --add safe.directory '*'
git config core.fileMode false
