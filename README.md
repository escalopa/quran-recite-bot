# 🕌 Quran Reading Bot

A Telegram bot that helps users practice Quran recitation by analyzing their voice recordings using AI. The bot provides detailed feedback on pronunciation accuracy and helps learners improve their Quran reading skills.

## ✨ Features

- 📖 **114 Surahs Support**: Browse and select from all Quran chapters
- 🌍 **Multi-language**: Supports English, Arabic, and Russian
- 🎯 **AI-Powered Analysis**: Get instant feedback on your recitation
- 📊 **Detailed Results**: Word-by-word analysis with operation codes (Correct, Substitution, Deletion, Insertion)
- 💾 **State Management**: Uses Redis FSM to track user progress
- 🎨 **User-friendly Interface**: Interactive keyboards for easy navigation
- 🐳 **Docker Support**: Easy deployment with Docker Compose

## 🏗️ Architecture

The project follows **Clean Architecture** and **Hexagonal Architecture** principles:

```
├── cmd/bot/              # Application entry point
├── internal/
│   ├── domain/          # Core business logic (entities, interfaces)
│   ├── application/     # Use cases and services
│   ├── adapter/         # External adapters
│   │   ├── telegram/    # Telegram bot implementation
│   │   ├── quranapi/    # Quran API client
│   │   ├── redis/       # Redis FSM storage
│   │   └── i18n/        # Internationalization
│   └── config/          # Configuration management
├── locales/             # Translation files (en, ar, ru)
└── docker/              # Docker configuration
```

### Layers:
- **Domain**: Pure business logic, no dependencies on frameworks
- **Application**: Use cases that orchestrate domain logic
- **Adapter**: Implementations of external interfaces (Telegram, Redis, API)

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or higher
- Redis (for local development)
- Docker and Docker Compose (for containerized deployment)
- Telegram Bot Token (from [@BotFather](https://t.me/botfather))
- Quran API access (endpoint and API key)

### Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/escalopa/quran-read-bot.git
   cd quran-read-bot
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Configure the bot**

   Create `config.yaml` from the example:
   ```bash
   cp config.example.yaml config.yaml
   ```

   Edit `config.yaml` and fill in your credentials:
   ```yaml
   telegram:
     token: "YOUR_TELEGRAM_BOT_TOKEN"

   redis:
     addr: "localhost:6379"
     password: ""
     db: 0

   quran_api:
     base_url: "https://quran.namaz.live"
     api_key: "YOUR_API_KEY"

   app:
     locales_dir: "locales"
     default_language: "en"
   ```

4. **Start Redis**
   ```bash
   docker run -d -p 6379:6379 redis:7-alpine
   ```

5. **Run the bot**
   ```bash
   make run
   # or
   go run ./cmd/bot/main.go
   ```

### Docker Deployment

1. **Create environment file**
   ```bash
   cp .env.example .env
   ```

   Edit `.env` with your credentials:
   ```env
   TELEGRAM_TOKEN=your_bot_token
   QURAN_API_URL=https://quran.namaz.live
   QURAN_API_KEY=your_api_key
   ```

2. **Start services**
   ```bash
   docker-compose up -d
   ```

3. **View logs**
   ```bash
   docker-compose logs -f bot
   ```

4. **Stop services**
   ```bash
   docker-compose down
   ```

## 📱 Usage

1. **Start the bot**: Send `/start` to your bot in Telegram
2. **Select language**: Choose your preferred language
3. **Choose Surah**: Browse and select a Surah from the list
4. **Enter Ayah number**: Type the verse number you want to practice
5. **Record**: Send your voice recording
6. **Get feedback**: Receive detailed analysis of your recitation

### Commands

- `/start` - Start the bot and select a Surah
- `/language` - Change the interface language
- `/help` - Display help information

## 🔧 Configuration

### Configuration File (`config.yaml`)

The bot can be configured using a YAML file or environment variables.

**Configuration precedence**: Environment variables > YAML file

### Environment Variables

- `TELEGRAM_TOKEN` - Telegram bot token
- `REDIS_ADDR` - Redis server address (default: localhost:6379)
- `REDIS_PASSWORD` - Redis password (optional)
- `QURAN_API_URL` - Quran API base URL
- `QURAN_API_KEY` - Quran API authentication key
- `CONFIG_PATH` - Path to config file (default: config.yaml)

## 🌍 Internationalization

The bot supports multiple languages. Translation files are located in the `locales/` directory:

- `en.yaml` - English
- `ar.yaml` - Arabic (العربية)
- `ru.yaml` - Russian (Русский)

To add a new language:
1. Create a new YAML file in `locales/` (e.g., `fr.yaml`)
2. Copy the structure from an existing file
3. Translate all message keys
4. Add the language to `domain.Language` constants

## 📊 API Integration

The bot integrates with the Quran Reading API (`quran.namaz.live`):

### Endpoints Used:
- `POST /recordings` - Submit voice recording for analysis
- `GET /recordings` - Retrieve recording results
- `GET /recordings/{learner_id}` - List user's recordings

### Audio Format:
- The API requires **WAV** format
- Telegram voice messages are in **OGG** format
- Consider using FFmpeg for audio conversion in production

### Response Format:
```json
{
  "recording_id": "uuid",
  "status": "done",
  "result": {
    "wer": 0.0,
    "ops": [
      {
        "ref_ar": "بِسْمِ",
        "hyp_ar": "بِسْمِ",
        "op": "C",
        "t_start": 0.0,
        "t_end": 0.5
      }
    ],
    "hypothesis": "transcribed text"
  }
}
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/domain/...
```

## 🛠️ Development

### Project Structure

```
quran-read-bot/
├── cmd/bot/                  # Main application
├── internal/
│   ├── domain/              # Business entities & interfaces
│   │   ├── entity.go        # Core entities (Surah, Ayah, Recording)
│   │   ├── port.go          # Interfaces (ports)
│   │   └── utils.go         # Helper functions
│   ├── application/         # Business logic
│   │   └── service.go       # Bot service implementation
│   ├── adapter/             # External implementations
│   │   ├── telegram/        # Telegram bot adapter
│   │   ├── quranapi/        # Quran API client
│   │   ├── redis/           # Redis FSM implementation
│   │   └── i18n/            # Internationalization
│   └── config/              # Configuration
├── locales/                 # Translation files
├── Dockerfile              # Container definition
├── docker-compose.yml      # Multi-container setup
└── Makefile               # Build automation
```

### Code Style

- Follow Go best practices and conventions
- Use `gofmt` for formatting
- Run linters before committing

```bash
make fmt
make lint
```

## 📝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Telegram Bot API](https://core.telegram.org/bots/api)
- [go-telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api)
- [go-redis](https://github.com/redis/go-redis)
- Quran Reading API

## 📞 Support

For questions or issues:
- Open an issue on GitHub
- Contact: [Your contact information]

---

Made with ❤️ for the Muslim community
