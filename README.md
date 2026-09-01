# Gophermart

Pet project сервиса накопительной системы магазина. 

## Схема БД

<img src="./docs/db_schema.png" alt="tg" alt="Medium" width="50%">

## Описание таблиц

В таблице users хранятся данные о пользователях. Она связана с другими таблицами связью 1 ко многим. В таблице orders информациях о заказах пользователя. Поле accrual (рассчитанные баллы) обновляется, когда мы получаем, что расчет заказа окончен. Таблица withdrawals - информация о списаниях средств. Здесь хранится история списаний. Для подсчета баланса баллов пользователей создал mat. view:

```sql
CREATE MATERIALIZED VIEW IF NOT EXISTS user_balance AS
SELECT
    u.id AS user_id,
    COALESCE(SUM(o.accrual), 0) - COALESCE(SUM(w.amount), 0) AS balance,
    COALESCE(SUM(o.accrual), 0) AS total_accrued,
    COALESCE(SUM(w.amount), 0) AS total_withdrawn
FROM users u
LEFT JOIN orders o ON u.id = o.user_id AND o.status = 'PROCESSED'  -- только подтверждённые заказы
LEFT JOIN withdrawals w ON u.id = w.user_id
GROUP BY u.id;
CREATE UNIQUE INDEX idx_user_balance_user_id ON user_balance (user_id);
```

Когда обновляю view:

1. Когда статус обработки заказа выполнен.
2. Когда произошло списание средств
3. Фоновое обновление кажде 30 секунд


С помощью горутин реализовал обновление view и обработку заказа.