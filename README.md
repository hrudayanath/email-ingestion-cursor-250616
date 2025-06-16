# Email Harvesting Microservice

A robust microservice for email harvesting, processing, and analysis built with Go, featuring OAuth integration with Gmail and Outlook, MongoDB storage, and LLM-powered analysis using Ollama.

## Features

- OAuth2 authentication for Gmail and Outlook
- Email harvesting and storage in MongoDB
- RESTful API endpoints for account and email management
- Email summarization and NER using local LLM (Ollama)
- Modern React-based UI client
- Docker containerization

## API Endpoints

### Account Management
- `POST /accounts` - Add Gmail or Outlook account (OAuth flow)
- `DELETE /accounts/{account_id}` - Remove an account

### Email Operations
- `GET /accounts/{account_id}/emails` - Fetch and store emails from external API
- `GET /emails` - List emails from local MongoDB
- `GET /emails/{id}` - Read a specific email from MongoDB
- `POST /emails/{id}/summarize` - Summarize a single email via Ollama
- `POST /emails/{id}/ner` - Perform NER using local LLM

## Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- MongoDB
- Ollama (for LLM operations)
- Node.js 18+ (for frontend development)

## Getting Started

1. Clone the repository
2. Set up environment variables (see `.env.example`)
3. Run with Docker Compose:
   ```bash
   docker-compose up --build
   ```

## Development

### Backend
```bash
cd backend
go mod download
go run cmd/server/main.go
```

### Frontend
```bash
cd frontend
npm install
npm run dev
```

## Environment Variables

Create a `.env` file with the following variables:

```env
# MongoDB
MONGODB_URI=mongodb://localhost:27017
MONGODB_DB=email_harvester

# OAuth
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
OUTLOOK_CLIENT_ID=your_outlook_client_id
OUTLOOK_CLIENT_SECRET=your_outlook_client_secret

# Server
PORT=8080
ENV=development

# Ollama
OLLAMA_API_URL=http://localhost:11434
```

## License

MIT 