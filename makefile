build:
	docker build -t reyshazni/gateway-proxy .

run:
	docker run -p 8080:8080 -p 80:80 reyshazni/gateway-proxy

push:
	docker build --platform=linux/amd64 -t reyshazni/gateway-proxy-amd . 
	docker push reyshazni/gateway-proxy-amd