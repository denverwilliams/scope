#!/bin/bash

docker build -t ilyanginx .
docker rm -f ilyanginx
docker run -dit --name ilyanginx -p 80:80 ilyaginx
