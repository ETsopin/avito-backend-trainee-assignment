# Переменные окружения
Сервис получает данные для подключения к бд из переменных окружения. Их необходимо вписать в .env файл. Достаточно указать значение целой строки для подключения к бд (POSTGRES_CONN), 
либо можно заполнить все остальные переменные, и сервис соберет строку самостоятельно.
# Запуск сервиса 
```
    docker-compose up --build app
```
В результате сервис должен отвечать по порту :8080
# Проверка работоспособности
Проверить работоспособность можно командой:
```
    curl -i localhost:8080/api/ping
```
В ответ должен прийти статус 200 и "ok" в body
Остальные запросы так же следуют структуре, которая дана в OpenAPI-файле в задании.
# Дополнительно
## "description" у предложений
Показалось странным, что при отправлении пользователю предложений или их списков в json нет поля "description", но решил следовать тому, что дано в openAPI, так что в моей реализации это поле тоже не отправляется.
## сущности в бд
Так как по заданию сущности пользователей и организации уже созданы, при запуске приложения они не создаются. При этом все остальные необходимые сущности, при условии, что они отсутствуют, создаются.
