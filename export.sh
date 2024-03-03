#! /bin/bash

set -x

cd /home/dom/repos/homeassistant-automate

source secret.sh

go run main.go daily-export 0

go run main.go graph

cp out.html ../blog/layouts/shortcodes/solar.html

cd /home/dom/repos/blog

./deploy.sh

cd /home/dom/repos/homeassistant-automate

go run main.go announce
