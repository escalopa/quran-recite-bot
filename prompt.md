# Quran Read Bot

Hello, I want to build a quran reading bot

## Stack

- Golang
- Clean architecture
- Hexagonal architecture
- Telegram bot library: https://github.com/go-telegram-bot-api/telegram-bot-api
- Use Redis for FSM
- Monolith application
- Docker for build and run
- Configuration a read from a yaml file or env

## Quran API

- Understanding what mistakes are in the recording sent by the user can be achieved by the external API from openapi.json which is, the client in code needs api endpoint and secret api token which are loaded from configuration file on run
- learner_id is the telegram chat_id
- ayah_id has the format `XXXYYY` where XXX is the number of Surah with leading zeros, and YYY the number of ayah in the surah with leading zeros
- File sent to the API must be a wav one
- You can find example of api requests to the api in api.md
- The backend of this api can be found under the path `~/work/follow_my_reading/backend`
- For the response show words of the resided ayah and OP for them

## Requirements

- User must be able to select Surah from a list of Surahs, the list must be a keyboard with buttons, each button is a surah, the button has the name and the number of the Surah
- Bot must support multiple languages, therfore all messages sent from it must be localized, use a yaml file with translation for each language
- After selecting the Surah the user get's a digits keyboard with the numbers from 0 to 9 with the layout of buttons as the telephone
- After that the user get's the oppourtunity to send a record with telegarm with the reciding of the ayah
- As a response he get's what the API returns in a JSON format message
