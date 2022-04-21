.PHONY: dredd

hooks/hooks: hooks/hooks.go
	cd hooks/; go build; cd ..;

server: main.go
	go build -o server

dredd: hooks/hooks server ../node_modules
	rm -f foo.db; touch foo.db; ../node_modules/.bin/dredd api.apib http://localhost:8080 --language go --server ./server --hookfiles hooks/hooks

../node_modules:
	cd ..; npm install
clean: 
	rm server hooks/hooks
