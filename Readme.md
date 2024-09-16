# Переменные окружения
Данные для базы данных берутся из переменных окружения. Перед запуском сервиса их неоебходимо записать в .env файл.
# Запуск сервиса 
```
    docker build -t app . && docker run -d -env-file .env -p 8080:8080 --name app-container app
```
В результате сервис должен отвечать по порту :8080
# Проверка работоспособности
Проверить работоспособность можно командой:
```
    curl -i localhost:8080/api/ping
```
В ответ должен прийти статус 200 и "ok" в body