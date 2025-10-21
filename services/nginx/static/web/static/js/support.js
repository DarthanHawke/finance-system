document.addEventListener('DOMContentLoaded', function() {
    const menuLinks = document.querySelectorAll('.menu a[data-section]');
    const supportSections = document.querySelectorAll('.support-section');
    let totalUsersCount = 0;

    // Функция для отображения первой буквы имени в аватаре
    function initAvatar() {
        const username = document.getElementById('username');
        const avatar = document.getElementById('userAvatar');
        
        if (username && avatar) {
            const firstLetter = username.textContent.charAt(0).toUpperCase();
            avatar.textContent = firstLetter;
        }
    }
    
    // Инициализируем аватар сразу при загрузке
    initAvatar();

    // Переключение между разделами панели поддержки
    menuLinks.forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            const sectionId = this.getAttribute('data-section');
            
            // Убираем активный класс у всех ссылок и разделов
            menuLinks.forEach(l => l.parentElement.classList.remove('active'));
            supportSections.forEach(s => s.classList.remove('active'));
            
            // Добавляем активный класс текущей ссылке и разделу
            this.parentElement.classList.add('active');
            document.getElementById(`${sectionId}-section`).classList.add('active');
            
            // Загружаем данные для активного раздела
            loadSectionData(sectionId);
        });
    });
    
    // Загрузка данных для активного раздела
    function loadSectionData(sectionId) {
        switch(sectionId) {
            case 'dashboard':
                loadDashboardData();
                break;
            case 'users':
                loadUsersData();
                break;
            case 'transactions':
                loadTransactionsData();
                break;
        }
    }
    
    const refreshStatsBtn = document.getElementById('refresh-stats-btn');
    if (refreshStatsBtn) {
        refreshStatsBtn.addEventListener('click', function(e) {
            e.preventDefault();
            refreshDashboardData();
        });
    }

    // Функция для обновления данных dashboard с визуальной обратной связью
    function refreshDashboardData() {
        const refreshStatsBtn = document.getElementById('refresh-stats-btn');
        const refreshTimer = document.getElementById('refresh-timer');
        
        // Показываем состояние загрузки
        if (refreshStatsBtn) {
            refreshStatsBtn.disabled = true;
            refreshStatsBtn.textContent = 'Обновление...';
        }
        
        if (refreshTimer) {
            refreshTimer.textContent = 'Идёт загрузка данных...';
        }
        
        // Сбрасываем таймер автообновления
        if (window.dashboardRefreshInterval) {
            clearInterval(window.dashboardRefreshInterval);
        }
        
        // Обновляем данные
        loadDashboardData()
            .finally(() => {
                // Восстанавливаем кнопку
                if (refreshStatsBtn) {
                    refreshStatsBtn.disabled = false;
                    refreshStatsBtn.textContent = 'Обновить данные';
                }
                
                if (refreshTimer) {
                    refreshTimer.textContent = 'Данные обновлены: ' + new Date().toLocaleTimeString();
                }
                
                // Запускаем таймер для следующего автообновления (например, каждые 5 минут)
                startDashboardAutoRefresh();
            });
    }

    // Функция для автоматического обновления данных
    function startDashboardAutoRefresh() {
        // Очищаем предыдущий интервал, если он был
        if (window.dashboardRefreshInterval) {
            clearInterval(window.dashboardRefreshInterval);
        }
        
        const refreshTimer = document.getElementById('refresh-timer');
        if (!refreshTimer) return;
        
        // Устанавливаем интервал обновления (5 минут = 300000 мс)
        const refreshInterval = 300000;
        let timeLeft = refreshInterval / 1000; // в секундах
        
        // Обновляем таймер сразу
        refreshTimer.textContent = `Следующее обновление через: ${timeLeft} сек.`;
        
        // Запускаем интервал
        window.dashboardRefreshInterval = setInterval(() => {
            timeLeft -= 1;
            
            if (timeLeft <= 0) {
                // Время вышло - обновляем данные
                refreshDashboardData();
            } else {
                // Обновляем таймер
                refreshTimer.textContent = `Следующее обновление через: ${timeLeft} сек.`;
            }
        }, 1000);
    }

    // Инициализируем автообновление при загрузке dashboard
    async function loadDashboardData() {
        // Загружаем всех пользователей без пагинации для дашборда
        try {
            const response = await fetch('/user/all', {
                method: 'GET',
                credentials: 'include'
            });
            const users = await response.json();
            
            // Сохраняем общее количество пользователей для пагинации
            totalUsersCount = users.length;
            document.getElementById('total-users').textContent = totalUsersCount;

            const usersMap = {};
            users.forEach(user => {
                usersMap[user.id] = user;
            });

            const [allAccounts, usersMap_1] = await Promise.all([
                Promise.all(users.map(user_1 => fetch(`/account/get?user_id=${user_1.id}`, {
                    method: 'GET',
                    credentials: 'include'
                })
                    .then(response_1 => response_1.json())
                    .then(accounts => {
                        return accounts.map(account => {
                            account.user_id = user_1.id;
                            return account;
                        });
                    })
                )),
                usersMap
            ]);

            const accounts_1 = [].concat(...allAccounts);
            const accountsMap = {};
            accounts_1.forEach(account_1 => {
                accountsMap[account_1.number] = {
                    user_id: account_1.user_id,
                    currency: account_1.currency
                };
            });

            document.getElementById('total-accounts').textContent = accounts_1.length;

            const allUserTransactions = await Promise.all(
                Object.keys(usersMap_1).map(userId => fetch(`/transactions/get?userID=${userId}`, {
                    method: 'GET',
                    credentials: 'include'
                })
                    .then(response_2 => response_2.json())
                    .then(transactions => {
                        const user_2 = usersMap_1[userId];
                        return transactions.map(transaction => {
                            const senderAccount = accountsMap[transaction.sender];
                            const receiverAccount = accountsMap[transaction.receiver];
                            const account_2 = senderAccount || receiverAccount;
                            const userFromAccount = account_2 ? usersMap_1[account_2.user_id] : null;

                            return {
                                ...transaction,
                                userName: userFromAccount ? userFromAccount.full_name : user_2.full_name,
                                type: 'transaction'
                            };
                        });
                    })
            ));

            const accountCreations = accounts_1.map(account_3 => {
                const user_3 = usersMap_1[account_3.user_id];
                return {
                    type: 'account_creation',
                    account_id: account_3.id,
                    currency: account_3.currency,
                    created_at: account_3.created_at,
                    userName: user_3 ? user_3.full_name : 'Неизвестный',
                    status: 'created'
                };
            });

            const allTransactions = [].concat(...allUserTransactions);
            const allActivities = [...allTransactions, ...accountCreations];
            allActivities.sort((a, b) => new Date(b.created_at) - new Date(a.created_at));

            document.getElementById('total-transactions').textContent = allTransactions.length;
            displayRecentActivities(allActivities.slice(0, 10));
        } catch (error) {
            console.error('Error loading dashboard data:', error);
            document.getElementById('activity-log').innerHTML =
                `<div class="error">Ошибка загрузки последних действий: ${error.message}</div>`;
        }
    }

    // При загрузке dashboard запускаем автообновление
    if (document.getElementById('dashboard-section')?.classList.contains('active')) {
        startDashboardAutoRefresh();
    }
    // Функция для отображения последних действий
    function displayRecentActivities(activities) {
        const activityLog = document.getElementById('activity-log');
        
        if (activities.length === 0) {
            activityLog.innerHTML = '<div class="empty">Нет данных о последних действиях</div>';
            return;
        }
        
        const activitiesHtml = activities.map(activity => {
            const dateStr = activity.created_at;
            const timestamp = dateStr ? new Date(dateStr) : null;
            
            const formattedDate = timestamp 
                ? timestamp.toLocaleDateString('ru-RU') + ' ' + timestamp.toLocaleTimeString('ru-RU')
                : 'Дата не указана';
            
            if (activity.type === 'account_creation') {
                return `
                    <div class="activity-item">
                        <div class="activity-time">${formattedDate}</div>
                        <div class="activity-text">
                            Пользователь <strong>${activity.userName}</strong> создал счет 
                            <strong>${activity.account_id}</strong> в валюте 
                            <strong>${activity.currency}</strong>.
                            Статус: <span class="status-created">создан</span>
                        </div>
                    </div>
                `;
            } else {
                return `
                    <div class="activity-item">
                        <div class="activity-time">${formattedDate}</div>
                        <div class="activity-text">
                            Пользователь <strong>${activity.userName}</strong> совершил платеж 
                            ${activity.sender ? `от ${activity.sender}` : ''}
                            ${activity.receiver ? `к ${activity.receiver}` : ''}
                            на сумму <strong>${activity.amount} ${activity.currency}</strong>.
                            Статус: <span class="status-${activity.status.toLowerCase()}">${activity.status}</span>
                        </div>
                    </div>
                `;
            }
        }).join('');
        
        activityLog.innerHTML = activitiesHtml;
    }
    
    const refreshUsersBtn = document.getElementById('refresh-users-btn');
    const prevPageBtn = document.getElementById('prev-page-btn');
    const nextPageBtn = document.getElementById('next-page-btn');
    const pageInfo = document.getElementById('page-info');

    let currentPage = 1;
    const usersPerPage = 10; 
    const searchInput = document.getElementById('search-user-input');
    const searchUserBtn = document.getElementById('search-user-btn');
    const clearSearchBtn = document.getElementById('clear-search-btn');

    // Обработчики для поиска
    if (searchUserBtn) {
        searchUserBtn.addEventListener('click', function(e) {
            e.preventDefault();
            searchUser();
        });
    }

    if (clearSearchBtn) {
        clearSearchBtn.addEventListener('click', function(e) {
            e.preventDefault();
            clearSearch();
        });
    }

    if (searchInput) {
        searchInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                searchUser();
            }
        });
    }

    function searchUser() {
        const userId = searchInput.value.trim();
        if (!userId) {
            alert('Пожалуйста, введите ID пользователя');
            return;
        }

        // Проверка на валидность UUID (можно убрать, если не нужно)
        const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
        if (!uuidRegex.test(userId)) {
            alert('Некорректный формат ID пользователя. Введите UUID.');
            return;
        }

        const usersTable = document.getElementById('users-table');
        const tbody = usersTable.querySelector('tbody');
        
        // Показываем состояние загрузки
        tbody.innerHTML = '<tr><td colspan="5" class="loading">Поиск пользователя...</td></tr>';
        
        // Отключаем кнопки
        if (searchUserBtn) searchUserBtn.disabled = true;
        if (clearSearchBtn) clearSearchBtn.disabled = true;
        
        fetch(`/user/profile?userID=${userId}`, {
            method: 'GET',
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) {
                if (response.status === 404) {
                    throw new Error('Пользователь не найден');
                } else {
                    throw new Error('Ошибка поиска пользователя');
                }
            }
            return response.json();
        })
        .then(user => {
            tbody.innerHTML = '';
            
            if (!user || !user.id) {
                tbody.innerHTML = '<tr><td colspan="5" class="empty">Пользователь не найден</td></tr>';
                return;
            }
            
            // Загружаем роли пользователя
            return fetch(`/role/users/${user.id}/relations`, {
                method: 'GET',
                credentials: 'include'
            })
            .then(response => {
                if (!response.ok) throw new Error('Ошибка загрузки ролей пользователя');
                return response.json();
            })
            .then(relations => {
                const roles = [];
                const relationTypes = relations.map(r => r.RelationType.toLowerCase());

                if (relationTypes.includes('admin')) roles.push('Админ');
                if (relationTypes.includes('support')) roles.push('Поддержка');
                if (relationTypes.includes('client')) roles.push('Клиент');

                if (roles.length === 0) {
                    roles.push('Заблокирован');
                }
                
                return { ...user, roles, relationTypes };
            });
        })
        .then(userWithRoles => {
            // Очищаем таблицу
            tbody.innerHTML = '';
            
            // Создаем строку для найденного пользователя
            const tr = document.createElement('tr');
            tr.classList.add('highlighted-user');
            
            // Создаем ячейку для действий
            const actionsTd = document.createElement('td');
            
            // Проверяем роли и добавляем соответствующие кнопки
            const isClient = userWithRoles.relationTypes.includes('client');
            const isSupport = userWithRoles.relationTypes.includes('support');
            const isAdmin = userWithRoles.relationTypes.includes('admin');
            
            if (isClient) {
                const accountsBtn = document.createElement('button');
                accountsBtn.className = 'action-btn accounts-user';
                accountsBtn.dataset.userId = userWithRoles.id;
                accountsBtn.textContent = 'Все счета';
                actionsTd.appendChild(accountsBtn);
                
                const editBtn = document.createElement('button');
                editBtn.className = 'action-btn edit-user';
                editBtn.dataset.userId = userWithRoles.id;
                editBtn.textContent = 'Редактировать';
                editBtn.addEventListener('click', () => showEditUserModal(userWithRoles));
                actionsTd.appendChild(editBtn);
            } else if (isSupport) {
                const editBtn = document.createElement('button');
                editBtn.className = 'action-btn edit-user';
                editBtn.dataset.userId = userWithRoles.id;
                editBtn.textContent = 'Редактировать';
                editBtn.addEventListener('click', () => showEditUserModal(userWithRoles));
                actionsTd.appendChild(editBtn);
            }
            
            tr.innerHTML = `
                <td>${userWithRoles.id}</td>
                <td>${userWithRoles.full_name || ''}</td>
                <td>${userWithRoles.email || ''}</td>
                <td>${userWithRoles.roles ? userWithRoles.roles.join(', ') : ''}</td>
            `;
            
            tr.appendChild(actionsTd);
            tbody.appendChild(tr);
        })
        .catch(error => {
            console.error('Ошибка поиска:', error);
            tbody.innerHTML = `<tr><td colspan="5" class="error">${error.message}</td></tr>`;
        })
        .finally(() => {
            if (searchUserBtn) searchUserBtn.disabled = false;
            if (clearSearchBtn) clearSearchBtn.disabled = false;
        });
    }

    function clearSearch() {
        if (searchInput) searchInput.value = '';
        currentPage = 1;
        loadUsersData();
    }

    
    if (refreshUsersBtn) {
        refreshUsersBtn.addEventListener('click', function(e) {
            e.preventDefault();
            currentPage = 1; // Сбрасываем на первую страницу при обновлении
            loadUsersData();
        });
    }

    if (prevPageBtn) {
        prevPageBtn.addEventListener('click', function(e) {
            e.preventDefault();
            if (currentPage > 1) {
                currentPage--;
                loadUsersData();
            }
        });
    }

    if (nextPageBtn) {
        nextPageBtn.addEventListener('click', function(e) {
            e.preventDefault();
            currentPage++;
            loadUsersData();
        });
    }

    function updatePaginationControls(totalUsers = totalUsersCount) {
        if (pageInfo) {
            const totalPages = Math.ceil(totalUsers / usersPerPage);
            pageInfo.textContent = `Страница ${currentPage} из ${totalPages}`;
        }
        
        if (prevPageBtn) {
            prevPageBtn.disabled = currentPage <= 1;
        }
        
        if (nextPageBtn) {
            const totalPages = Math.ceil(totalUsers / usersPerPage);
            nextPageBtn.disabled = currentPage >= totalPages;
        }
    }

    function loadUsersData() {
    const usersTable = document.getElementById('users-table');
    const refreshBtn = document.getElementById('refresh-users-btn');
    const tbody = usersTable.querySelector('tbody');
    if (searchInput) searchInput.value = '';
    // Показываем состояние загрузки
    tbody.innerHTML = '<tr><td colspan="5" class="loading">Загрузка пользователей...</td></tr>';
    
    // Отключаем все кнопки управления
    if (refreshBtn) refreshBtn.disabled = true;
    if (prevPageBtn) prevPageBtn.disabled = true;
    if (nextPageBtn) nextPageBtn.disabled = true;
    
    if (refreshBtn) {
        refreshBtn.textContent = 'Загрузка...';
        refreshBtn.classList.add('loading');
    }
    
    // Рассчитываем offset для пагинации
    const offset = (currentPage - 1) * usersPerPage;
    
    // Загружаем пользователей с пагинацией
    fetch(`/user/all?limit=${usersPerPage}&offset=${offset}`, {
        method: 'GET',
        credentials: 'include'
    })
    .then(response => {
        if (!response.ok) throw new Error('Ошибка загрузки пользователей');
        return response.json();
    })
    .then(users => {
        tbody.innerHTML = '';
        
        if (users.length === 0) {
            tbody.innerHTML = '<tr><td colspan="5" class="empty">Нет пользователей</td></tr>';
            if (currentPage > 1) {
                currentPage--;
                loadUsersData();
            }
            return;
        }
        
        // Обновляем пагинацию с сохраненным общим количеством
        updatePaginationControls(totalUsersCount);
            
            // Для каждого пользователя загружаем его роли
            const userPromises = users.map(async user => {
                try {
                    const response = await fetch(`/role/users/${user.id}/relations`, {
                        method: 'GET',
                        credentials: 'include'
                    });
                    if (!response.ok) throw new Error('Ошибка загрузки ролей пользователя');
                    const relations = await response.json();
                    // Определяем роли на основе relations (приводим к lowercase для надежности)
                    const roles = [];
                    const relationTypes = relations.map(r => r.RelationType.toLowerCase());

                    if (relationTypes.includes('admin')) roles.push('Админ');
                    if (relationTypes.includes('support')) roles.push('Поддержка');
                    if (relationTypes.includes('client')) roles.push('Клиент');

                    // Если нет ни одной из стандартных ролей
                    if (roles.length === 0) {
                        roles.push('Заблокирован');
                    }
                    return { ...user, roles, relationTypes };
                } catch (error) {
                    console.error(`Ошибка загрузки ролей для пользователя ${user.id}:`, error);
                    return { ...user, roles: ['Ошибка загрузки ролей'], relationTypes: [] };
                }
            });
            
            // Ждем загрузки ролей для всех пользователей
            return Promise.all(userPromises);
        })
        .then(usersWithRoles => {
            // Отображаем пользователей с их ролями
            usersWithRoles.forEach(user => {
                const tr = document.createElement('tr');
                
                // Создаем ячейку для действий
                const actionsTd = document.createElement('td');
                
                // Проверяем роли и добавляем соответствующие кнопки
                const isClient = user.relationTypes.includes('client');
                const isSupport = user.relationTypes.includes('support');
                const isAdmin = user.relationTypes.includes('admin');
                
                if (isClient) {
                    const accountsBtn = document.createElement('button');
                    accountsBtn.className = 'action-btn accounts-user';
                    accountsBtn.dataset.userId = user.id;
                    accountsBtn.textContent = 'Все счета';
                    actionsTd.appendChild(accountsBtn);
                }
                tr.innerHTML = `
                    <td>${user.id}</td>
                    <td>${user.full_name}</td>
                    <td>${user.email}</td>
                    <td>${user.roles ? user.roles.join(', ') : ''}</td>
                `;
                
                tr.appendChild(actionsTd);
                tbody.appendChild(tr);
            });
        })
        .catch(error => {
        console.error('Ошибка загрузки пользователей:', error);
        tbody.innerHTML = `<tr><td colspan="5" class="error">${error.message}</td></tr>`;
        currentPage = 1;
        updatePaginationControls();
    })
    .finally(() => {
        if (refreshBtn) {
            refreshBtn.disabled = false;
            refreshBtn.textContent = 'Обновить список';
            refreshBtn.classList.remove('loading');
        }
        updatePaginationControls(totalUsersCount);
    });
}

    // Добавляем обработчики для кнопок "Все счета"
    document.addEventListener('click', function(e) {
        if (e.target && e.target.classList.contains('accounts-user')) {
            const userId = e.target.dataset.userId;
            showUserAccountsModal(userId);
        }
    });

    // Переменные для хранения состояния операций
    let currentAccountId = null;
    let currentOperationsPage = 1;
    const operationsPerPage = 10;

    // Функция для показа модального окна со счетами пользователя
    function showUserAccountsModal(userId) {
        const modal = document.getElementById('accounts-modal');
        const title = document.getElementById('accounts-modal-title');
        const accountsContainer = document.getElementById('accounts-list-container');
        
        // Устанавливаем заголовок
        title.textContent = `Счета пользователя #${userId}`;
        
        // Показываем состояние загрузки
        accountsContainer.innerHTML = '<div class="loading">Загрузка счетов...</div>';
        
        // Показываем модальное окно
        modal.classList.add('active');
        
        // Загружаем счета пользователя
        fetch(`/account/get?user_id=${userId}`, {
            method: 'GET',
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка загрузки счетов');
            return response.json();
        })
        .then(accounts => {
            if (!accounts || accounts.length === 0) {
                accountsContainer.innerHTML = '<div class="empty">У пользователя нет счетов</div>';
                return;
            }
            
            // Создаем HTML для списка счетов
            let html = '<div class="accounts-list">';
            
            accounts.forEach(account => {
                const availableBalance = account.balance - account.blocked_amount;
                
                html += `
                    <div class="account-card">
                        <div class="account-name">${account.account_name || 'Без названия'}</div>
                        <div class="account-id">ID: ${account.id}</div>
                        <div class="account-balance">${availableBalance.toFixed(2)}</div>
                        <div class="account-currency">${account.currency}</div>
                        <div class="account-actions">
                            <button class="view-operations-btn" data-account-id="${account.id}">
                                Все операции
                            </button>
                        </div>
                    </div>
                `;
            });
            
            html += '</div>';
            accountsContainer.innerHTML = html;
            
            // Добавляем обработчики для кнопок "Все операции"
            document.querySelectorAll('.view-operations-btn').forEach(btn => {
                btn.addEventListener('click', function() {
                    currentAccountId = this.dataset.accountId;
                    currentOperationsPage = 1;
                    showAccountOperations(currentAccountId, currentOperationsPage);
                });
            });
        })
        .catch(error => {
            console.error('Ошибка загрузки счетов:', error);
            accountsContainer.innerHTML = `<div class="error">${error.message}</div>`;
        });
        
        // Обработчик закрытия модального окна
        const closeBtn = modal.querySelector('.close-btn');
        closeBtn.addEventListener('click', () => {
            modal.classList.remove('active');
            // Возвращаемся к списку счетов при закрытии
            document.getElementById('accounts-list-container').style.display = 'block';
            document.getElementById('operations-list-container').style.display = 'none';
        });
        
        // Закрытие при клике вне модального окна
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.classList.remove('active');
                // Возвращаемся к списку счетов при закрытии
                document.getElementById('accounts-list-container').style.display = 'block';
                document.getElementById('operations-list-container').style.display = 'none';
            }
        });
        
        // Обработчик кнопки "Назад к счетам"
        document.getElementById('back-to-accounts-btn').addEventListener('click', function() {
            document.getElementById('accounts-list-container').style.display = 'block';
            document.getElementById('operations-list-container').style.display = 'none';
        });
        
        // Обработчики пагинации операций
        document.getElementById('prev-operations-btn').addEventListener('click', function() {
            if (currentOperationsPage > 1) {
                currentOperationsPage--;
                showAccountOperations(currentAccountId, currentOperationsPage);
            }
        });
        
        document.getElementById('next-operations-btn').addEventListener('click', function() {
            currentOperationsPage++;
            showAccountOperations(currentAccountId, currentOperationsPage);
        });
    }

    // Функция для показа операций по счету
    function showAccountOperations(accountId, page = 1) {
        const operationsContainer = document.getElementById('operations-list');
        const operationsTitle = document.getElementById('operations-title');
        const pageInfo = document.getElementById('operations-page-info');
        
        // Показываем состояние загрузки
        operationsContainer.innerHTML = '<div class="loading">Загрузка операций...</div>';
        
        // Рассчитываем offset для пагинации
        const offset = (page - 1) * operationsPerPage;
        
        // Загружаем операции
        fetch(`/transactions/history?account_id=${accountId}&limit=${operationsPerPage}&offset=${offset}`, {
            method: 'GET',
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка загрузки операций');
            return response.json();
        })
        .then(operations => {
            // Если страница пустая и это не первая страница, возвращаемся на предыдущую
            if (operations.length === 0 && page > 1) {
                currentOperationsPage--;
                showAccountOperations(accountId, currentOperationsPage);
                return;
            }
            
            // Обновляем информацию о странице
            pageInfo.textContent = `Страница ${page}`;
            
            // Обновляем кнопки пагинации
            document.getElementById('prev-operations-btn').disabled = page <= 1;
            document.getElementById('next-operations-btn').disabled = operations.length < operationsPerPage;
            
            // Устанавливаем заголовок
            operationsTitle.textContent = `Операции по счету #${accountId}`;
            
            // Если операций нет
            if (operations.length === 0) {
                operationsContainer.innerHTML = '<div class="empty">Нет операций по этому счету</div>';
                return;
            }
            
            // Создаем HTML для списка операций
            let html = '';
            
            operations.forEach(operation => {
                const amountClass = operation.amount >= 0 ? 'positive' : 'negative';
                const amountSign = operation.amount >= 0 ? '+' : '';
                const processedAt = operation.processed_at ? new Date(operation.processed_at).toLocaleString() : 'Не обработано';
                
                html += `
                    <div class="operation-item">
                        <div class="operation-amount ${amountClass}">${amountSign}${operation.amount.toFixed(2)} ${operation.currency}</div>
                        <div class="operation-details">
                            <span class="operation-type">${operation.operation_type}</span>
                            <span class="operation-status ${operation.status.toLowerCase()}">${operation.status}</span>
                            ${operation.description ? `<div>${operation.description}</div>` : ''}
                        </div>
                        <div class="operation-date">${new Date(operation.created_at).toLocaleString()} (обработано: ${processedAt})</div>
                    </div>
                `;
            });
            
            operationsContainer.innerHTML = html;
            
            // Переключаемся на список операций
            document.getElementById('accounts-list-container').style.display = 'none';
            document.getElementById('operations-list-container').style.display = 'block';
        })
        .catch(error => {
            console.error('Ошибка загрузки операций:', error);
            operationsContainer.innerHTML = `<div class="error">${error.message}</div>`;
            
            // Если ошибка, возвращаемся к списку счетов
            document.getElementById('accounts-list-container').style.display = 'block';
            document.getElementById('operations-list-container').style.display = 'none';
        });
    }    
    
   // Объявляем все необходимые функции в начале

// Вспомогательная функция для получения класса статуса
function getStatusClass(status) {
    switch(status) {
        case 'completed': return 'status-completed';
        case 'failed': return 'status-failed';
        case 'refunded': return 'status-refunded';
        default: return 'status-pending';
    }
}

// Функция для добавления пагинации
function addTransactionsPagination(transactionsTable, currentPage, totalPages) {
    const paginationContainer = document.createElement('div');
    paginationContainer.className = 'table-pagination';
    
    // Кнопка "Назад"
    const prevBtn = document.createElement('button');
    prevBtn.className = 'pagination-btn';
    prevBtn.id = 'prev-transactions-btn';
    prevBtn.disabled = currentPage <= 1;
    prevBtn.innerHTML = `
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="15 18 9 12 15 6"></polyline>
        </svg>
        Назад
    `;
    prevBtn.addEventListener('click', () => {
        if (currentPage > 1) {
            displayTransactionsPage(currentPage - 1);
        }
    });
    
    // Информация о странице
    const pageInfo = document.createElement('span');
    pageInfo.id = 'transactions-page-info';
    pageInfo.textContent = `Страница ${currentPage} из ${totalPages}`;
    
    // Кнопка "Вперед"
    const nextBtn = document.createElement('button');
    nextBtn.className = 'pagination-btn';
    nextBtn.id = 'next-transactions-btn';
    nextBtn.disabled = currentPage >= totalPages;
    nextBtn.innerHTML = `
        Вперед
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="9 18 15 12 9 6"></polyline>
        </svg>
    `;
    nextBtn.addEventListener('click', () => {
        if (currentPage < totalPages) {
            displayTransactionsPage(currentPage + 1);
        }
    });
    
    // Добавляем элементы в контейнер
    paginationContainer.appendChild(prevBtn);
    paginationContainer.appendChild(pageInfo);
    paginationContainer.appendChild(nextBtn);
    
    // Удаляем старую пагинацию, если есть
    const oldPagination = transactionsTable.parentNode.querySelector('.table-pagination');
    if (oldPagination) oldPagination.remove();
    
    // Добавляем пагинацию после таблицы
    transactionsTable.parentNode.appendChild(paginationContainer);
}

// Функция для показа деталей платежа из кеша
function showTransactionDetailsFromCache(transactionId) {
    if (!window.filteredTransactions) {
        alert('Данные о платежах не загружены');
        return;
    }
    
    const transaction = window.filteredTransactions.find(p => p.id === transactionId);
    if (!transaction) {
        alert('Платеж не найден в загруженных данных');
        return;
    }
    
    // Создаем модальное окно
    const modal = document.createElement('div');
    modal.className = 'modal';
    modal.innerHTML = `
        <div class="modal-content">
            <span class="close-btn">&times;</span>
            <h2>Детали платежа #${transaction.id}</h2>
            
            <div class="transaction-details">
                <div class="detail-row">
                    <span class="detail-label">Отправитель:</span>
                    <span class="detail-value">${transaction.sender || 'Не указан'}</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Получатель:</span>
                    <span class="detail-value">${transaction.receiver || 'Не указан'}</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Сумма:</span>
                    <span class="detail-value">${transaction.amount.toFixed(2)} ${transaction.currency}</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Статус:</span>
                    <span class="detail-value status-badge ${transaction.statusClass}">${transaction.status}</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Дата:</span>
                    <span class="detail-value">${transaction.formattedDate}</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Описание:</span>
                    <span class="detail-value">${transaction.description || 'Нет описания'}</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Пользователь:</span>
                    <span class="detail-value">${transaction.user?.full_name || transaction.user?.email || 'Неизвестный пользователь'}</span>
                </div>
            </div>
        </div>
    `;
    
    // Добавляем модальное окно в DOM
    document.body.appendChild(modal);
    
    // Обработчик закрытия
    const closeBtn = modal.querySelector('.close-btn');
    closeBtn.addEventListener('click', () => {
        modal.remove();
    });
    
    // Закрытие при клике вне модального окна
    modal.addEventListener('click', (e) => {
        if (e.target === modal) {
            modal.remove();
        }
    });
}

const transactionsSection = document.getElementById('transactions-section');
transactionsSection.insertAdjacentHTML('afterbegin', `
    <div class="search-container">
        <input type="text" id="search-transaction-input" placeholder="Поиск по ID платежа или пользователя">
        <button id="search-transaction-btn">Найти</button>
        <button id="clear-transaction-search">Сбросить</button>
    </div>
`);

// Функция поиска платежей
function searchTransactions(query) {
    if (!query || !window.filteredTransactions) return window.filteredTransactions || [];
    
    const lowerQuery = query.toLowerCase();
    return window.filteredTransactions.filter(transaction => {
        return (
            transaction.id.toLowerCase().includes(lowerQuery) ||
            (transaction.user?.id.toLowerCase().includes(lowerQuery)) ||
            (transaction.user?.email?.toLowerCase().includes(lowerQuery)) ||
            (transaction.user?.full_name?.toLowerCase().includes(lowerQuery))
        );
    });
}

// Обработчики поиска
document.getElementById('search-transaction-btn').addEventListener('click', () => {
    const query = document.getElementById('search-transaction-input').value.trim();
    if (!query) return;
    
    const results = searchTransactions(query);
    if (results.length === 0) {
        alert('Платежи не найдены');
        return;
    }
    
    window.filteredTransactions = results;
    displayTransactionsPage(1);
});

document.getElementById('clear-transaction-search').addEventListener('click', () => {
    document.getElementById('search-transaction-input').value = '';
    loadTransactionsData(); // Перезагружаем исходные данные
});

// Модифицированная функция отображения страницы
function displayTransactionsPage(page) {
    const transactionsTable = document.getElementById('transactions-table');
    const tbody = transactionsTable.querySelector('tbody');
    const transactionsPerPage = 10;
    
    if (!window.filteredTransactions || window.filteredTransactions.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" class="empty">Нет платежей</td></tr>';
        
        // Удаляем пагинацию если нет данных
        const oldPagination = transactionsTable.parentNode.querySelector('.table-pagination');
        if (oldPagination) oldPagination.remove();
        return;
    }
    
    // Рассчитываем индексы для текущей страницы
    const startIndex = (page - 1) * transactionsPerPage;
    const endIndex = startIndex + transactionsPerPage;
    const pageTransactions = window.filteredTransactions.slice(startIndex, endIndex);
    
    // Очищаем таблицу
    tbody.innerHTML = '';
    
    // Заполняем таблицу платежами текущей страницы
    pageTransactions.forEach(transaction => {
        const tr = document.createElement('tr');
        
        tr.innerHTML = `
            <td>${transaction.id}</td>
            <td>${transaction.sender || 'Не указан'}</td>
            <td>${transaction.receiver || 'Не указан'}</td>
            <td>${transaction.amount.toFixed(2)} ${transaction.currency}</td>
            <td><span class="status-badge ${transaction.statusClass}">${transaction.status}</span></td>
            <td>${transaction.formattedDate}</td>
            <td class="actions-cell">
                <button class="action-btn info-btn" data-transaction-id="${transaction.id}">Детали</button>
            </td>
        `;
        tbody.appendChild(tr);
    });
    
    // Добавляем пагинацию
    addTransactionsPagination(transactionsTable, page, Math.ceil(window.filteredTransactions.length / transactionsPerPage));
    
    // Добавляем обработчики для кнопок деталей
    document.querySelectorAll('.action-btn[data-transaction-id]').forEach(btn => {
        btn.addEventListener('click', function() {
            const transactionId = this.getAttribute('data-transaction-id');
            showTransactionDetailsFromCache(transactionId);
        });
    });
}

// Основная функция загрузки платежей
function loadTransactionsData() {
    const transactionsTable = document.getElementById('transactions-table');
    const tbody = transactionsTable.querySelector('tbody');
    tbody.innerHTML = '<tr><td colspan="7" class="loading">Загрузка платежей...</td></tr>';
    
    // Получаем значения фильтров
    const status = document.getElementById('transaction-status').value;
    const dateRange = document.getElementById('transaction-date').value;
    
    // Сначала загружаем всех пользователей
    fetch('/user/all', {
        method: 'GET',
        credentials: 'include'
    })
    .then(response => {
        if (!response.ok) throw new Error('Ошибка загрузки пользователей');
        return response.json();
    })
    .then(users => {
        // Для каждого пользователя загружаем его платежи
        const transactionPromises = users.map(user => {
            return fetch(`/transactions/get?userID=${user.id}`, {
                method: 'GET',
                credentials: 'include'
            })
            .then(response => {
                if (!response.ok) return [];
                return response.json();
            })
            .then(transactions => {
                return transactions.map(p => ({ 
                    ...p, 
                    user,
                    formattedDate: new Date(p.created_at || p.timestamp).toLocaleString(),
                    statusClass: getStatusClass(p.status)
                }));
            })
            .catch(() => []); // Игнорируем ошибки для отдельных пользователей
        });
        
        return Promise.all(transactionPromises);
    })
    .then(allUserTransactions => {
        // Объединяем все платежи в один массив
        const allTransactions = [].concat(...allUserTransactions);
        
        // Применяем фильтрацию
        let filteredTransactions = [...allTransactions];
        
        // Фильтр по статусу
        if (status !== 'all') {
            filteredTransactions = filteredTransactions.filter(p => p.status === status);
        }
        
        // Фильтр по дате
        const now = new Date();
        filteredTransactions = filteredTransactions.filter(p => {
            const transactionDate = new Date(p.created_at || p.timestamp);
            
            switch(dateRange) {
                case 'today':
                    return transactionDate.toDateString() === now.toDateString();
                case 'week':
                    const weekAgo = new Date(now);
                    weekAgo.setDate(weekAgo.getDate() - 7);
                    return transactionDate >= weekAgo;
                case 'month':
                    const monthAgo = new Date(now);
                    monthAgo.setMonth(monthAgo.getMonth() - 1);
                    return transactionDate >= monthAgo;
                default:
                    return true;
            }
        });
        
        // Сортируем по дате (новые сначала)
        filteredTransactions.sort((a, b) => {
            const dateA = new Date(a.created_at || a.timestamp);
            const dateB = new Date(b.created_at || b.timestamp);
            return dateB - dateA;
        });
        
        // Сохраняем отфильтрованные платежи для пагинации и деталей
        window.filteredTransactions = filteredTransactions;
        
        // Отображаем первую страницу
        displayTransactionsPage(1);
    })
    .catch(error => {
        console.error('Ошибка загрузки платежей:', error);
        tbody.innerHTML = `<tr><td colspan="7" class="error">${error.message}</td></tr>`;
    });
}

// Добавляем обработчик для кнопки применения фильтров
document.getElementById('apply-filters').addEventListener('click', () => {
    loadTransactionsData();
});
    
    // Загрузка данных при первом открытии
    const activeSection = document.querySelector('.support-section.active');
    if (activeSection) {
        loadSectionData(activeSection.id.replace('-section', ''));
    }

    const logoutBtn = document.getElementById('logout-btn');
    if (logoutBtn) {
        logoutBtn.addEventListener('click', function(e) {
            e.preventDefault();
            
            fetch('/auth/logout', {
                method: 'POST',
                credentials: 'include'
            })
            .then(response => {
                if (!response.ok) throw new Error('Ошибка выхода');
                return response.json();
            })
            .then(data => {
                window.location.href = '/auth';
            })
            .catch(error => {
                alert(error.message);
            });
        });
    }
    // Проверка аутентификации при загрузке страницы
    checkAuth();
     // Функция проверки роли и перенаправления
    function checkUserRoleAndRedirect() {
        fetch('/role/role', {
            method: 'GET',
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка проверки роли');
            return response.json();
        })
        .then(data => {
            // Получаем роль из ответа
            const role = data.role;
            
            // Перенаправляем в зависимости от роли
            if (role === 'admin') {
                window.location.href = '/admin';
            } else if (role === 'support') {
                //пропускаем
            } else {
                window.location.href = '/accounts';
            }
        });
    }
    // Функция проверки аутентификации
    function checkAuth() {
        fetch('/user/profile', {
            method: 'GET',
            credentials: 'include'
        })        
        .then(response => {
            if (response.ok) {
                // Если пользователь аутентифицирован, перенаправляем
                checkUserRoleAndRedirect();
            } else if (!response.ok) {
                // Если пользователь не аутентифицирован, перенаправляем на страницу входа
                window.location.href = '/auth';
            }
        })
        .catch(error => {
            console.error('Auth check failed:', error);
            window.location.href = '/auth';
        });
    }
});