# E-Nav 开发指南

## 环境准备

### Docker 安装

#### Ubuntu/Debian
```bash
# 卸载旧版本
sudo apt-get remove docker docker-engine docker.io containerd runc

# 更新软件包
sudo apt-get update

# 安装依赖
sudo apt-get install \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

# 添加 Docker 官方 GPG 密钥
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

# 设置稳定版仓库
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# 安装 Docker Engine
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io
```

#### CentOS/RHEL
```bash
# 卸载旧版本
sudo yum remove docker \
                docker-client \
                docker-client-latest \
                docker-common \
                docker-latest \
                docker-latest-logrotate \
                docker-logrotate \
                docker-engine

# 安装依赖
sudo yum install -y yum-utils

# 设置仓库
sudo yum-config-manager \
    --add-repo \
    https://download.docker.com/linux/centos/docker-ce.repo

# 安装 Docker Engine
sudo yum install docker-ce docker-ce-cli containerd.io
```

### Docker Compose 安装

#### Linux（通用方法）
```bash
# 下载最新版本
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose

# 添加执行权限
sudo chmod +x /usr/local/bin/docker-compose
```

## 项目构建与推送

### 克隆源代码
```bash
git clone https://github.com/ecouus/E-Nav.git
cd E-Nav
```

### 重建并推送镜像
```bash
# 停止并删除旧容器
docker stop e-nav
docker rm e-nav

# 删除旧镜像
docker rmi ecouus/e-nav:latest

# 重新构建（不使用缓存）
docker build --no-cache -t ecouus/e-nav:latest .

# 登录 Docker Hub
docker login

# 推送到 Docker Hub
docker push ecouus/e-nav:latest
```

## 注意事项
- 安装后可能需要重启系统或添加用户到 docker 组
- 国内服务器建议配置 Docker 镜像加速
- 构建前检查 Dockerfile 配置
- 推送需要对 Docker Hub 仓库有写入权限
