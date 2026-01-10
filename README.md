# go-aliyunmc 🎮

A semi-automated Minecraft server management system built with Go and Alibaba Cloud SDK. This backend service provides comprehensive control over Minecraft server deployment, management, and monitoring on Alibaba Cloud infrastructure.

## 🚀 Overview

Go-AliyunMC is a sophisticated server management system that leverages Alibaba Cloud's ECS (Elastic Compute Service) to provide an automated solution for Minecraft server hosting. It allows users to create, manage, and monitor Minecraft servers through a RESTful API interface with built-in monitoring capabilities.

The system features automatic server scaling, real-time status monitoring, automated backups to Alibaba Cloud Object Storage Service (OSS), and seamless deployment workflows.

## ✨ Features

- **Instance Management**: Create, delete, and monitor Alibaba Cloud ECS instances for Minecraft servers
- **Real-time Monitoring**: Continuous server status, player count, and online player list tracking
- **Automated Backups**: Scheduled world backups stored in Alibaba Cloud OSS with retention policies
- **Server Archiving**: Full server archiving and restoration capabilities
- **Authentication System**: JWT-based authentication for secure API access
- **Task Management**: Asynchronous task execution with cancellation support
- **Auto-scaling**: Automatic instance status monitoring and management
- **Public IP Management**: Automatic public IP allocation and management
- **Command Execution**: Remote server command execution (start/stop/backup/archive)

## 🏗️ Architecture

The system is organized into several key modules:

### Core Modules

- **`clients`**: Alibaba Cloud SDK clients for ECS and VPC services
- **`config`**: Application configuration management with TOML-based settings
- **`handlers`**: REST API endpoints organized by functionality
- **`monitors`**: Real-time monitoring services for instances and server status
- **`helpers`**: Utility functions including database operations and command execution
- **`globals`**: Global variables and cached Alibaba Cloud resources

### API Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/instance/:instanceId` | Get specific instance information | No |
| GET | `/active-or-latest-instance` | Get active or latest instance | No |
| GET | `/instance-status` | Get active instance status | No |
| POST | `/instance` | Create new ECS instance | ✅ |
| DELETE | `/instance/:instanceId` | Delete specific instance | ✅ |
| GET | `/instance-deploy` | Deploy instance configuration | ✅ |
| POST | `/auth/token` | User authentication and token generation | No |
| GET | `/stream` | Server event streaming | ✅ |
| GET | `/server/info` | Get Minecraft server status | ✅ |
| GET | `/server/exec` | Execute server commands | ✅ |
| GET | `/server/backups` | Get backup information | ✅ |

### Monitoring Services

The system runs several background monitors:

- **Active Instance Monitor**: Tracks instance lifecycle and status
- **Server Status Monitor**: Monitors Minecraft server health and player activity
- **Public IP Allocator**: Manages public IP assignment to instances
- **Backup Monitor**: Handles automated backup processes

## 🛠️ Configuration

The system uses a `config.toml` file for configuration with the following sections:

### Server Configuration
```toml
[server]
expose = 8080          # Port to expose the API server on
jwt_secret = "..."     # Secret key for JWT token signing
```

### Alibaba Cloud Configuration
```toml
[aliyun]
region_id = "..."              # Alibaba Cloud region ID
access_key_id = "..."          # Access key ID
access_key_secret = "..."      # Access key secret

[aliyun.ecs]
internet_max_bandwidth_out = 100  # Maximum bandwidth for public network
image_id = "..."                  # ECS image ID for instances
hostname = "mc-server"            # Hostname for created instances
security_group_id = "..."         # Security group for instances
```

### Database Configuration
```toml
[database]
host = "localhost"     # Database host
port = 3306           # Database port
username = "..."      # Database username
password = "..."      # Database password
database = "..."      # Database name
```

### Deployment Configuration
```toml
[deploy]
packages = ["screen", "zip", "unzip"]  # Additional packages to install
ssh_public_key = "..."                 # SSH public key for access
java_version = 11                      # Java version to install (Zulu)
oss_root = "oss://bucket-name"        # Alibaba Cloud OSS bucket root
backup_path = "backups"                # Backup directory in OSS
archive_path = "archives"              # Archive directory in OSS
```

## 📦 Deployment Process

The system uses templated shell scripts for automated server deployment:

1. **User Creation**: Creates a dedicated user for running the Minecraft server
2. **SSH Configuration**: Sets up SSH access with provided public key
3. **Repository Configuration**: Configures optimized package repositories
4. **Java Installation**: Installs Zulu Java runtime
5. **System Utilities**: Installs required system packages
6. **OSS Integration**: Configures Alibaba Cloud OSS utility
7. **Storage Setup**: Formats and mounts data disks for server files
8. **Archive Restoration**: Copies server files from configured archives

## 🔒 Security Features

- JWT-based authentication for all management endpoints
- Secure credential management using Alibaba Cloud SDK
- SSH key-based server access
- Database connection pooling with security best practices
- CORS configuration for web interface support

## 🔄 Backup & Recovery

The system provides comprehensive backup and recovery capabilities:

- **Automatic Backups**: Scheduled world backups to Alibaba Cloud OSS
- **Backup Retention**: Configurable number of backup copies to retain
- **Archive Management**: Full server archiving for complete state preservation
- **Quick Restoration**: Automated restoration from backup/Archive files

## 📊 Monitoring & Status

Real-time monitoring includes:

- Minecraft server running status
- Player count tracking
- Online player list
- Instance health status
- Network connectivity monitoring

## 🚀 Getting Started

1. **Prerequisites**:
   - Go 1.24 or higher
   - Alibaba Cloud account with appropriate permissions
   - MySQL database
   - Alibaba Cloud OSS bucket

2. **Configuration**:
   - Create `config.toml` with your settings
   - Set up database schema
   - Configure Alibaba Cloud credentials

3. **Build and Run**:
   ```bash
   go build -o go-aliyunmc .
   ./go-aliyunmc
   ```

4. **API Access**:
   - Server will start on configured port (default 8080)
   - API documentation available at `/docs` (if implemented)
   - Authentication required for management endpoints

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

For support, please open an issue in the repository or contact the maintainers directly.