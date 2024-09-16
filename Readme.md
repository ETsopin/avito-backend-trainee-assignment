# Переменные окружения
Данные для базы данных берутся из переменных окружения. Перед запуском сервиса их неоебходимо записать в .env файл.
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
Показалось странным, что при отправлении пользователю предложений или их списков в json нет поля "description", но решил следовать тому, что дано в openAPI, так что в моей реализации это поле тоже не отправляется.