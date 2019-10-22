export PS1='\e[1m\e[31m[\h] \e[32m($(docker-prompt)) \e[34m\u@$(hostname -i)\e[35m \w\e[0m\n$ '
alias vi='vim'
export PATH=$PATH:/root/go/bin
export DOCKER_HOST=""
cat /etc/motd
echo $BASHPID > /var/run/cwd
