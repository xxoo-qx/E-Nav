#!/bin/bash

# 颜色定义
RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
BLUE="\033[34m"
NC="\033[0m"

# 项目信息
REPO_URL="https://github.com/ecouus/E-Nav.git"
INSTALL_DIR="/root/E-Nav"
SERVICE_NAME="E-Nav"

# 检查命令行参数
if [ $# -eq 0 ]; then
    echo -e "${RED}请指定操作: install 或 uninstall${NC}"
    echo -e "使用方法:"
    echo -e "  ${GREEN}./One-Click.sh install${NC}   - 安装 E-Nav"
    echo -e "  ${GREEN}./One-Click.sh uninstall${NC} - 卸载 E-Nav"
    echo -e ""
    echo -e "快速安装命令:"
    echo -e "  ${GREEN}curl -fsSL https://raw.githubusercontent.com/ecouus/E-Nav/main/One-Click.sh -o One-Click.sh && chmod +x One-Click.sh && bash One-Click.sh install${NC}"
    echo -e ""
    echo -e "快速卸载命令:"
    echo -e "  ${GREEN}curl -fsSL https://raw.githubusercontent.com/ecouus/E-Nav/main/One-Click.sh -o One-Click.sh && chmod +x One-Click.sh && bash One-Click.sh uninstall${NC}"
    exit 1
fi

# 检查是否为root用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}请使用root用户运行此脚本${NC}"
    exit 1
fi

# 安装函数
install() {
    echo -e "${GREEN}开始安装 E-Nav...${NC}"

    # 检查并安装必要的软件
    echo -e "${YELLOW}检查并安装必要的软件...${NC}"
    if ! command -v git &> /dev/null; then
        apt-get update
        apt-get install -y git
    fi

    # 检查并安装Go
    if ! command -v go &> /dev/null; then
        echo -e "${YELLOW}未检测到Go,开始安装...${NC}"
        wget https://go.dev/dl/go1.24.1.linux-amd64.tar.gz
        tar -C /usr/local -xzf go1.24.1.linux-amd64.tar.gz
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc
        export PATH=$PATH:/usr/local/go/bin
        rm go1.24.1.linux-amd64.tar.gz
    fi

    # 克隆项目
    echo -e "${GREEN}克隆项目...${NC}"
    cd /root
    if [ -d "$INSTALL_DIR" ]; then
        echo -e "${YELLOW}检测到已存在的安装，备份数据...${NC}"
        if [ -d "$INSTALL_DIR/data" ]; then
            mkdir -p /root/E-Nav-backup
            cp -r $INSTALL_DIR/data /root/E-Nav-backup/
            echo -e "${GREEN}数据已备份到 /root/E-Nav-backup/data${NC}"
        fi
        rm -rf $INSTALL_DIR
    fi
    
    git clone $REPO_URL $INSTALL_DIR
    cd $INSTALL_DIR

    # 确保数据目录存在
    echo -e "${GREEN}初始化数据目录...${NC}"
    mkdir -p data

    # 恢复之前的数据（如果存在）
    if [ -d "/root/E-Nav-backup/data" ]; then
        echo -e "${GREEN}恢复之前的数据...${NC}"
        cp -r /root/E-Nav-backup/data/* $INSTALL_DIR/data/
        rm -rf /root/E-Nav-backup
    fi
    
    # 初始化Go项目
    echo -e "${GREEN}初始化Go项目...${NC}"
    go mod init E-Nav
    go mod tidy
    go get github.com/gorilla/mux
    go get github.com/gorilla/sessions
    go get golang.org/x/crypto/bcrypt

    # 编译项目
    echo -e "${GREEN}编译项目...${NC}"
    go build -o E-Nav

    # 创建系统服务
    echo -e "${GREEN}创建系统服务...${NC}"
    cat > /etc/systemd/system/$SERVICE_NAME.service << EOF
[Unit]
Description=E-Nav Go Web Application
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/E-Nav
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable $SERVICE_NAME
    systemctl start $SERVICE_NAME

    # 检查服务状态
    if systemctl is-active --quiet $SERVICE_NAME; then
        echo -e "${GREEN}E-Nav 安装成功!${NC}"
        echo -e "${GREEN}您可以通过访问 http://服务器IP:1239 来使用导航站${NC}"
        echo -e "${BLUE}重要信息:${NC}"
        echo -e "${BLUE}1. 所有数据存储在 $INSTALL_DIR/data 目录${NC}"
        echo -e "${BLUE}2. 如需备份，只需备份该目录下的文件即可${NC}"
        echo -e "${BLUE}3. 默认管理员密码为: admin${NC}"
    else
        echo -e "${RED}E-Nav 安装失败,请检查日志${NC}"
        systemctl status $SERVICE_NAME
        exit 1
    fi
}

# 卸载函数
uninstall() {
    echo -e "${YELLOW}开始卸载 E-Nav...${NC}"
    
    # 停止并禁用服务
    if systemctl is-active --quiet $SERVICE_NAME; then
        echo -e "${YELLOW}停止并禁用 E-Nav 服务...${NC}"
        systemctl stop $SERVICE_NAME
        systemctl disable $SERVICE_NAME
    fi
    
    # 询问是否保留数据
    echo -e "${YELLOW}是否保留配置和数据? [y/N]${NC}"
    read -r keep_data
    
    # 删除服务文件
    echo -e "${YELLOW}删除服务文件...${NC}"
    rm -f /etc/systemd/system/$SERVICE_NAME.service
    systemctl daemon-reload
    
    # 备份数据（如果用户选择保留）
    if [[ "$keep_data" =~ ^[Yy]$ ]] && [ -d "$INSTALL_DIR/data" ]; then
        echo -e "${GREEN}备份数据到 /root/E-Nav-data-backup...${NC}"
        mkdir -p /root/E-Nav-data-backup
        cp -r $INSTALL_DIR/data/* /root/E-Nav-data-backup/
        echo -e "${GREEN}数据已备份到 /root/E-Nav-data-backup${NC}"
    fi
    
    # 删除项目文件
    echo -e "${YELLOW}删除项目文件...${NC}"
    rm -rf $INSTALL_DIR
    
    # 询问是否删除Go环境
    if [[ ! "$keep_data" =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}是否删除Go环境? [y/N]${NC}"
        read -r remove_go
        
        if [[ "$remove_go" =~ ^[Yy]$ ]]; then
            echo -e "${YELLOW}清理Go环境...${NC}"
            rm -rf /usr/local/go
            sed -i '/\/usr\/local\/go\/bin/d' /root/.bashrc
        fi
        
        # 询问是否删除git
        echo -e "${YELLOW}是否卸载git? [y/N]${NC}"
        read -r remove_git
        
        if [[ "$remove_git" =~ ^[Yy]$ ]]; then
            echo -e "${YELLOW}卸载git...${NC}"
            apt remove --purge -y git
            apt autoremove -y
            apt clean
        fi
    fi
    
    echo -e "${GREEN}E-Nav 已卸载${NC}"
    
    if [[ "$keep_data" =~ ^[Yy]$ ]]; then
        echo -e "${GREEN}数据已备份到 /root/E-Nav-data-backup${NC}"
        echo -e "${GREEN}再次安装时可将此目录中的文件复制到新安装的data目录中${NC}"
    fi
}

# 根据参数执行相应操作
case "$1" in
    "install")
        install
        ;;
    "uninstall")
        uninstall
        ;;
    *)
        echo -e "${RED}无效的参数: $1${NC}"
        echo -e "使用方法:"
        echo -e "  ${GREEN}./One-Click.sh install${NC}   - 安装 E-Nav"
        echo -e "  ${GREEN}./One-Click.sh uninstall${NC} - 卸载 E-Nav"
        exit 1
        ;;
esac
