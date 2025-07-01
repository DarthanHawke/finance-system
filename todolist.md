Todo
Это я вкратце решил описать последовательность своих задачек, воть:
Сегодня/завтра
1. В LogoutAll зачем-то userid
2. в sso string string токены, надо обернуть в структуру
3. При GetAllPayments из Billing-service :2025-07-01T14:30:52.425Z ERROR payment/payment.go:150 cant get payment {"component": "billing_service", "error": "storage.billing.GetByUser: pq: relation \"users\" does not exist"}
4. Blacklist токенов(redis)
5. Дописать UpdateStatus, Cansel for billing-service 
Завтра/послезавтра
6. клиент Billing в Client-Service, 
7. Handlers для клиента, 
8. HTML для 
Потом(В приоритете)
9. Client SSO-role в Billing
10. Service слой для управления ролями в Billing
11. Server Handlers для ролей в Billing
12. Скрипт/миграция для создания суперпользователя в БД с ролью админа
13. ХОТЯБЫ ПАРУ ТЕСТОВ НА ВСЕ СЕРВИСЫ
Потом(в среднем приоритете)
14. WebSocet сервис: Client-WebSocet-Billing в виде чат-бота
15. Redis кэш для SSO
16. Поправить логи в Billing в репо и в сервисном слоях, тоже самое в Client и в ChatBot, если надо
17. Допилить везде комментарии
18. Покрытие тестов 40%
Потом(в нижнем приорите)
19. Покрытие тестов 60%
20.  Добавить функциональности в Notification-service
Никогда(или когда-нибудь, но уже не обязательно)
21. Покрытие тестов 80%
22. Grafana
23. Docker-secrets, конфиги для Prod, CI/CD - GitHub Actions
24. Kubernetes(Если его вообще возможно заюзать)
