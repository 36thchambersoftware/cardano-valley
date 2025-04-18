###
# Deploy on Prod
###

buildProd:
	sudo systemctl stop cardano-valley.service
	cd ~/git/cardano-valley
	rm /usr/local/bin/cardano-valley
	rm cardano-valley
	go build -o cardano-valley
	sudo cp -p cardano-valley /usr/local/bin/.
	sudo systemd-analyze verify cardano-valley.service
	sudo systemctl daemon-reload
	sudo systemctl start cardano-valley.service
	sudo journalctl -f -u cardano-valley