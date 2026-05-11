# Практическая работа №10 (семестр 2)

## Выполнила: Сорокина К.С., ЭФМО-01-25

## Тема: Горизонтальное масштабирование: использование Load Balancer (NGINX)

### Цель:

Освоить базовый подход к горизонтальному масштабированию backend-приложения за счёт запуска нескольких экземпляров одного сервиса и распределения входящих HTTP-запросов через NGINX в роли балансировщика нагрузки.

## Технологии

- **Go** — язык реализации сервиса
- **NGINX 1.27** — балансировщик нагрузки (load balancer)
- **Docker Compose** — оркестрация контейнеров
- **round-robin** — алгоритм балансировки запросов

## Структура проекта

```
PR10_sem2/
├── services/
│   └── tasks/
│       ├── cmd/
│       │   └── server/
│       │       └── main.go     ← сервис с /health, /v1/tasks, X-Instance-ID
│       ├── go.mod
│       └── Dockerfile
└── deploy/
    └── lb/
        ├── nginx.conf          ← конфигурация балансировщика
        └── docker-compose.yml  ← две реплики tasks + nginx
```

---

## Архитектура стенда

```
Клиент (Postman)
       │
       ▼
  NGINX :8080          ← единая точка входа
       │
  upstream tasks_backend
  ┌────┴────┐
  │         │
tasks_1   tasks_2      ← реплики сервиса (без внешних портов)
:8082     :8082
```

Клиент взаимодействует только с NGINX на порту 8080. Реплики tasks_1 и tasks_2 не пробрасываются наружу и доступны только внутри Docker-сети.

---

## Конфигурация NGINX (deploy/lb/nginx.conf)

```nginx
events {}

http {
    upstream tasks_backend {
        server tasks_1:8082;
        server tasks_2:8082;
    }

    server {
        listen 8080;

        location / {
            proxy_pass http://tasks_backend;
            proxy_set_header Host $host;
            proxy_set_header X-Request-ID $request_id;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header Authorization $http_authorization;
        }
    }
}
```

`upstream tasks_backend` описывает группу backend-реплик. NGINX использует алгоритм round-robin по умолчанию — запросы поочерёдно направляются на tasks_1 и tasks_2. Имена сервисов резолвятся через внутренний DNS Docker.

---

## Идентификация инстанса: X-Instance-ID

Каждая реплика читает переменную окружения `INSTANCE_ID` и добавляет её как заголовок в каждый ответ:

```go
instanceID := os.Getenv("INSTANCE_ID")
w.Header().Set("X-Instance-ID", instanceID)
```

Это позволяет наглядно видеть в Postman (вкладка Headers ответа), какая именно реплика обработала запрос.

---

## Docker Compose (deploy/lb/docker-compose.yml)

```yaml
version: "3.9"

services:
  tasks_1:
    build:
      context: ../../services/tasks
    container_name: tasks_1
    environment:
      APP_PORT: "8082"
      INSTANCE_ID: "tasks-1"

  tasks_2:
    build:
      context: ../../services/tasks
    container_name: tasks_2
    environment:
      APP_PORT: "8082"
      INSTANCE_ID: "tasks-2"

  nginx:
    image: nginx:1.27-alpine
    container_name: nginx_lb
    ports:
      - "8080:8080"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - tasks_1
      - tasks_2
```

---

## Запуск

```bash
docker compose -f deploy/lb/docker-compose.yml up -d --build
```

Проверка статуса:
```bash
docker compose -f deploy/lb/docker-compose.yml ps
```

![](https://github.com/krrristina/PR10_sem2/blob/main/screenshots/проверка%20работы%20.png)

## Проверки в Postman

### 1. Health endpoint
- **Method:** `GET`
- **URL:** `http://localhost:8080/health`

Ответ: `{"instance":"tasks-1","status":"ok"}`
Заголовок ответа: `X-Instance-ID: tasks-1`

![](https://github.com/krrristina/PR10_sem2/blob/main/screenshots/health%20endpoint.png)

![](https://github.com/krrristina/PR10_sem2/blob/main/screenshots/health%20endpoint%202.png)

### 2. Балансировка запросов
- **Method:** `GET`
- **URL:** `http://localhost:8080/v1/tasks`

При повторных запросах заголовок `X-Instance-ID` чередуется между `tasks-1` и `tasks-2` — подтверждает работу round-robin балансировки.

![](https://github.com/krrristina/PR10_sem2/blob/main/screenshots/tasks-1.png)

![](https://github.com/krrristina/PR10_sem2/blob/main/screenshots/tasks-2.png)

### 3. Проброс заголовков через NGINX
- **Method:** `GET`
- **URL:** `http://localhost:8080/v1/tasks`
- **Header:** `Authorization: Bearer demo-token`

NGINX корректно передаёт заголовок `Authorization` в backend.

![](https://github.com/krrristina/PR10_sem2/blob/main/screenshots/проброс%20заголовков%20через%20NGINX.png)

### 4. Отказоустойчивость — остановка реплики
```bash
docker compose -f deploy/lb/docker-compose.yml stop tasks_1
```

После остановки `tasks_1` все запросы обслуживает `tasks_2`. Заголовок `X-Instance-ID: tasks-2` во всех ответах. Система продолжает работать.

![](https://github.com/krrristina/PR10_sem2/blob/main/screenshots/остановка%20одного%20контейнера.png)

![](https://github.com/krrristina/PR10_sem2/blob/main/screenshots/только%20tasks-2%20после%20остановки.png)

### 5. Возврат реплики в строй
```bash
docker compose -f deploy/lb/docker-compose.yml start tasks_1
```
![](https://github.com/krrristina/PR10_sem2/blob/main/screenshots/повторный%20запуск%20tasks-1.png)

Балансировка снова идёт между двумя репликами.

## Ответы на контрольные вопросы

**Что такое горизонтальное масштабирование?**
Увеличение мощности системы за счёт запуска нескольких экземпляров одного приложения. В этой работе запускаются tasks_1 и tasks_2 — две реплики одного сервиса.

**Чем оно отличается от вертикального масштабирования?**
Вертикальное — больше CPU, RAM, мощнее сервер. Имеет физический предел. Горизонтальное — больше экземпляров приложения. Требует балансировщика, но позволяет масштабироваться практически неограниченно.

**Зачем нужен load balancer?**
Без балансировщика клиент должен знать адреса всех реплик и сам выбирать, к какой обращаться. Load balancer скрывает внутреннюю структуру и даёт единую точку входа, распределяя нагрузку автоматически.

**Какую роль в этой работе выполняет NGINX?**
Принимает все входящие запросы на порту 8080, выбирает одну из реплик по алгоритму round-robin и проксирует запрос туда. Клиент не знает, сколько реплик работает внутри.

**Что такое upstream в NGINX?**
Именованная группа backend-серверов, между которыми NGINX распределяет трафик. В данной работе — tasks_backend, содержащий tasks_1:8082 и tasks_2:8082.

**Почему для горизонтального масштабирования желательно делать сервис stateless?**
Если сервис хранит данные в памяти конкретного процесса, разные реплики будут видеть разные данные. Stateless-сервис держит все данные в общей БД или Redis, доступных всем репликам одинаково.

**Зачем нужен health endpoint?**
Позволяет быстро проверить доступность конкретного экземпляра. В production используется балансировщиками для автоматического исключения недоступных реплик из ротации.

**Почему полезно добавлять X-Instance-ID?**
Делает балансировку видимой и проверяемой. Без этого заголовка невозможно доказать, что запросы реально распределяются между репликами.

**Что происходит при остановке одной из реплик?**
NGINX продолжает получать запросы и направляет их только на доступную реплику. Система деградирует по мощности, но не отказывает.

**Почему клиенту удобнее работать с одной точкой входа?**
Клиент не обязан знать сколько реплик запущено и какие у них адреса. Всё это скрыто за единым адресом NGINX. При добавлении новых реплик клиент ничего не меняет.
