#!/usr/bin/env bash
set -euo pipefail

echo "==> Actualizando paquetes del sistema..."
sudo apt update && sudo apt upgrade -y

echo "==> Instalando dependencias básicas..."
sudo apt install -y ca-certificates curl gnupg lsb-release ufw

echo "==> Instalando Docker Engine..."
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo \"$VERSION_CODENAME\") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

echo "==> Habilitando y arrancando Docker..."
sudo systemctl enable docker
sudo systemctl start docker

echo "==> Configurando UFW (firewall)..."
sudo ufw allow OpenSSH
sudo ufw allow http
sudo ufw allow https
echo "y" | sudo ufw enable || true

echo "==> Creando estructura de directorios para el ERP..."
sudo mkdir -p /opt/invorya-erp
sudo chown "$USER":"$USER" /opt/invorya-erp

echo "==> Setup inicial completado. Copia tu código a /opt/invorya-erp y ejecuta:"
echo "    docker compose -f docker-compose.prod.yml up -d --build"

