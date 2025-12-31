#!/bin/bash

echo "--> Installing Docker"
sudo yum update -y
sudo yum install docker -y

echo "--> Installing Docker-compose"
sudo curl -SL https://github.com/docker/compose/releases/download/v2.20.3/docker-compose-linux-x86_64 -o /usr/local/bin/docker-compose
sudo ln -s /usr/local/bin/docker-compose /usr/bin/docker-compose
sudo chmod 755 /usr/local/bin/docker-compose

echo "--> configuring as a service"
sudo usermod -a -G docker ec2-user
sudo systemctl enable docker.service

echo "--> Starting Docker"
sudo service docker start

echo "--> Installing ngrok"
wget https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-linux-amd64.tgz
sudo tar xvzf ./ngrok-v3-stable-linux-amd64.tgz -C /usr/local/bin

echo "--> Getting Configs"
wget https://raw.githubusercontent.com/emanor-okta/go-scim/refs/heads/main/docker-compose.yml
wget https://raw.githubusercontent.com/emanor-okta/go-scim/refs/heads/main/config.yaml

echo "--> Creating Volume setting scripts"
docker volume create goscim-config
goscim_config_destination=`docker volume inspect goscim-config | grep Mountpoint | awk -F'"' '{print $4}'`
sudo cp config.yaml $goscim_config_destination
echo "alias vigoscim2='sudo vi ${goscim_config_destination}/config.yaml'" >> ~/.bash_profile

echo "--> done.."
echo ""

echo "Login to ngrok (https://dashboard.ngrok.com/login) and get an API Token"
echo -e "Run '\e[32mngrok config add-authtoken <token>\e[0m' to configure ngrok"
echo ""
echo -e "Start ngrok with '\e[32mngrok http 8082\e[0m'"
echo -e "Open another terminal window and start docker with '\e[32mdocker-compose up\e[0m'"

echo ""
