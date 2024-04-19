build:
	docker build -t reyshazni/sidecar-proxy .

run:
	docker run -p 8080:8080 -p 80:80 reyshazni/sidecar-proxy

push:
	docker build --platform=linux/amd64 -t reyshazni/sidecar-proxy-amd . 
	docker push reyshazni/sidecar-proxy-amd