# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cardano Valley is a Discord bot that provides Cardano staking and farming functionality. The bot allows Discord communities to set up reward systems based on Discord roles and Cardano asset holdings, distributing rewards to users automatically.

## Architecture

### Core Components

- **main.go**: Entry point that initializes Discord bot, MongoDB connection, and command handlers
- **pkg/discord/**: Discord bot implementation with command handlers, role/holder reward systems
- **pkg/cardano/**: Cardano CLI wrapper for blockchain operations  
- **pkg/blockfrost/**: Blockfrost API client for Cardano blockchain data
- **pkg/cv/**: Core business logic including config, user management, and utilities
- **pkg/db/**: MongoDB database connection and operations
- **pkg/logger/**: Structured logging utilities

### Key Features

- **Discord Commands**: Slash commands for user registration, wallet linking, reward management, airdrops
- **Reward Systems**: 
  - Role-based rewards: Users with specific Discord roles earn rewards daily
  - Holder-based rewards: Users holding specific Cardano assets earn proportional rewards
- **Wallet Management**: Users can link Cardano payment addresses to their Discord accounts
- **Airdrop System**: Create and distribute token airdrops to users

### Data Flow

1. Users register via Discord commands and link Cardano wallets
2. Daily reward cycles check user roles and asset holdings via Blockfrost API
3. Rewards are credited to user accounts in MongoDB
4. Users can withdraw accumulated rewards

## Commands

### Build and Deployment

```bash
# Build the application
go build -o cardano-valley

# Production deployment (requires sudo/systemctl setup)
make buildProd
```

### Development

```bash
# Run locally
go run main.go

# Get dependencies
go mod tidy

# Run tests (requires GO_TESTING=true to skip Discord API initialization)
GO_TESTING=true go test ./pkg/discord -v

# Run specific test suites
GO_TESTING=true go test ./pkg/discord -v -run "Airdrop"
GO_TESTING=true go test ./pkg/discord -v -run "TestHelper"
GO_TESTING=true go test ./pkg/discord -v -run "Recovery"
GO_TESTING=true go test ./pkg/discord -v -run "RealWorld"
```

## Environment Variables

Required environment variables:
- `CARDANO_VALLEY_TOKEN`: Discord bot token
- `CARDANO_VALLEY_APPLICATION_ID`: Discord application ID
- `BLOCKFROST_PROJECT_ID`: Blockfrost API project ID
- `DISCORD_WEBHOOK_URL`: Discord webhook URL for notifications
- MongoDB connection variables (configured in pkg/db)

## Database Schema

Uses MongoDB with collections:
- `configs`: Server/guild configurations and reward settings
- `users`: User profiles, linked wallets, and reward balances
- `command-history`: Audit log of slash commands
- `modal-history`: Audit log of Discord modal interactions
- `component-history`: Audit log of Discord component interactions

## Discord Integration

- **Commands**: Defined in `pkg/discord/command-*.go` files
- **Handlers**: Command logic in corresponding handler functions
- **Modals**: Interactive forms for complex user input
- **Components**: Select menus and buttons for user interaction
- **Lockout System**: Prevents concurrent command execution per user

## Cardano Integration

- **Blockfrost API**: Primary method for reading blockchain data
- **Cardano CLI**: Direct CLI calls for advanced operations (path: `/usr/local/bin/cardano-cli`)
- **ADA Handles**: Automatic conversion of $handle format to Cardano addresses

## Code Patterns

- All Discord interactions log to command/modal/component history collections
- User lockout prevents concurrent operations per Discord user
- Reward cycles run automatically at scheduled times (daily at 00:00 and 12:00 UTC)
- Context with timeouts used for all external API calls
- Structured logging throughout with contextual information