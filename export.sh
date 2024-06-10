#! /bin/bash

set -x

cd /home/dom/repos/homeassistant-automate

source secret.sh

/usr/local/go/bin/go run main.go daily-export 0

git add data.csv
git commit -m "Daily upload $(date)"
git push origin main

/usr/local/go/bin/go run main.go graph

cp out.html ../blog/layouts/shortcodes/solar.html

cd /home/dom/repos/blog

./deploy.sh

cd /home/dom/repos/homeassistant-automate

/usr/local/go/bin/go run main.go announce
