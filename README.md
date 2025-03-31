# mattermost_bot
[![wakatime](https://wakatime.com/badge/user/018badf6-44ca-4a0f-82e9-9b27db08764a/project/c0105f89-e7d5-4ae3-9e01-6ac48f295a06.svg)](https://wakatime.com/badge/user/018badf6-44ca-4a0f-82e9-9b27db08764a/project/c0105f89-e7d5-4ae3-9e01-6ac48f295a06)  
Проект представляет собой бота, который реализует команды для создания опросов, голосования и получения результатов в Mattermost

<details>
<summary>Техническое задание</summary>
Необходимо добавить функционал для системы голосования внутри
чатов мессенджера Mattermost. Бот должен позволять пользователям
создавать голосования, голосовать за предложенные варианты и
просматривать результаты.   

Функциональные требования:

1. Создание голосования (Бот регистрирует голосование и
возвращает сообщение с ID голосования и вариантами ответов).
2. Голосование (Пользователь отправляет команду, указывая ID
голосования и вариант ответа).
3. Просмотр результатов (Любой пользователь может запросить
текущие результаты голосования).
4. Завершение голосования (Создатель голосования может
завершить его досрочно).
5. Удаление голосования (Возможность удаления голосования).

Нефункциональные требования:
- Код должен быть написан на Go.
- Логирование действий.
- Хранение данных в Tarantool.
- Использование dokker и dokker-compose для поднятия и
развертывания dev-среды.
- Код должен быть выложен на github или аналог. Код должен быть
сопровожден инструкцией по сборке и установке.
</details>

## Команды
#### `/poll create "question" "option1" "option2" "optionN"`
- создает опрос с заданным вопросом и вариантами ответа.  
возвращает ID опроса и варианты ответов.  
>**Poll ID**: 784337a5  
**Question**: _you're a bot?_  
**Options**:  
  [1] _yes_  
  [2] _no_
#### `/poll vote poll_id choice_id` 
- записывает голос пользователя за указанный ID ответа

#### `/poll result poll_id`
- возвращает результаты голосования по указанному ID опроса 
>**Question**: _you're a bot?_  
    [1] votes: **1** (_yes_)  
    [2] votes: **0** (_no_)  
#### `/poll end poll_id`
- завершает опрос
#### `/poll delete poll_id`
- удаляет опрос
#### `/poll help`
- выводит список доступных команд   
>i know only this command:  
`/poll create "question" "option1" "option2" "optionN"`  
`/poll vote poll_id choice_id`  
`/poll result poll_id`  
`/poll end poll_id`  
`/poll delete poll_id`  
`/poll help`

## Требования

- [Go](https://go.dev/doc/install)
- [Docker](https://docs.docker.com/get-docker/) и [Docker Compose](https://docs.docker.com/compose/install/)
- [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)

## Настройка
1. Клонируйте репозиторий:
```bash
git clone https://github.com/jaam8/mattermost_bot.git
```

2. Скопируйте файл `.env.example` и настройте переменные окружения по своему усмотрению:
```bash
cp .env.example .env
```

| Переменная           | Значение по умолчанию | Описание                              |
|----------------------|-----------------------|---------------------------------------|
| `TARANTOOL_USER`     | `admin`               | Логин пользователя базы данных        |
| `TARANTOOL_PASSWORD` | `secret`              | Пароль пользователя базы данных       |
| `TARANTOOL_HOST`     | `localhost`           | Хост базы данных                      |
| `TARANTOOL_PORT`     | `3301`                | Порт базы данных                      |
| `LOG_LEVEL`          | `info`                | Уровень логирования (`debug`, `info`) |
| `MM_WS_URL`          |                       | Mattermost URL по WebSocket (`ws://`) |
| `MM_URL`             |                       | Mattermost URL по HTTP   (`http://`)  |
| `BOT_TOKEN`          |                       | Токен доступа к боту в Mattermost     |

## Запуск с Docker

Для сборки и запуска контейнеров выполните:

```bash
docker-compose build
docker-compose up
```

Для остановки контейнеров выполните:

```bash
docker-compose down
```

## Структура проекта

```
    mattermost_bot
    ├── cmd              # Точка входа в приложение
    ├── internal         
    │   ├── api          # хендлеры для работы с API
    │   ├── config       # Инциализация env переменных
    │   ├── models       # Модели данных
    │   ├── repository   # Логика работы с БД
    │   └── service      # Бизнес-логика
    ├── pkg
    │   ├── logger       # Логирование
    │   └── tarantool    # Tarantool клиент
    ├── tarantool        # Конфиги и миграции
    │   ├── config.yml
    │   ├── init.lua
    │   └── instances.yml
    ├── Dockerfile       # Dockerfile для сборки контейнера
    ├── docker-compose.yml   
    ├── .gitignore       
    ├── .env.example     # Шаблон env файла
    └── README.md        # Документация
```

## Миграции

При старте бота автоматически запускаются миграции БД.  

