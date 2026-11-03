#!/bin/sh
set -eu

deploy_key="${1:?usage: sh setup-server.sh \"<deploy public key>\"}"
app_dir=/opt/bingusbot

# A dedicated user runs the bot and receives deploys.
if ! id bingus >/dev/null 2>&1; then
	useradd --system --create-home --shell /bin/sh bingus
fi
# So failed deploys can show the bot's logs.
usermod -aG systemd-journal bingus

install -d -o bingus -g bingus -m 755 "$app_dir"
install -d -o bingus -g bingus -m 700 "$app_dir/data"

install -d -o bingus -g bingus -m 700 /home/bingus/.ssh
printf '%s\n' "$deploy_key" >/home/bingus/.ssh/authorized_keys
chown bingus:bingus /home/bingus/.ssh/authorized_keys
chmod 600 /home/bingus/.ssh/authorized_keys

# Let deploys restart the bot, and nothing else, without a password.
printf 'bingus ALL=(root) NOPASSWD: /usr/bin/systemctl restart bingusbot\n' >/etc/sudoers.d/bingusbot
chmod 440 /etc/sudoers.d/bingusbot
visudo -cf /etc/sudoers.d/bingusbot

install -m 644 "$(dirname "$0")/bingusbot.service" /etc/systemd/system/bingusbot.service
systemctl daemon-reload
systemctl enable bingusbot

echo "Done"
