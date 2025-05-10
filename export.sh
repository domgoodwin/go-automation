#! /bin/bash

set -x

cd /home/dom/repos/domgoodwin/go-automation

source secret.sh

/home/dom/.nix-profile/bin/go run main.go daily-export 0

git add data.csv
git commit -m "Daily upload $(date)"
git push origin main

/home/dom/.nix-profile/bin/go run main.go graph

cp out.html ../blog/layouts/shortcodes/solar.html

cd /home/dom/repos/domgoodwin/blog

./deploy.sh

cd /home/dom/repos/domgoodwin/go-automation

# /home/dom/.nix-profile/bin/go run main.go announce
