document.addEventListener('DOMContentLoaded', function() {
    const menuLinks = document.querySelectorAll('.menu a[data-section]');
    const adminSections = document.querySelectorAll('.admin-section');
    const tabBtns = document.querySelectorAll('.roles-tabs .tab-btn');
    const tabContents = document.querySelectorAll('.roles-tabs .tab-content');
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
    // Переключение между разделами админ-панели
    menuLinks.forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            const sectionId = this.getAttribute('data-section');
            
            // Убираем активный класс у всех ссылок и разделов
            menuLinks.forEach(l => l.parentElement.classList.remove('active'));
            adminSections.forEach(s => s.classList.remove('active'));
            
            // Добавляем активный класс текущей ссылке и разделу
            this.parentElement.classList.add('active');
            document.getElementById(`${sectionId}-section`).classList.add('active');
            
            // Загружаем данные для активного раздела
            loadSectionData(sectionId);
            
            // Если открываем раздел пользователей и общее количество еще не известно
            if (sectionId === 'users' && totalUsersCount === 0) {
                // Сначала загружаем данные для дашборда, чтобы получить общее количество
                fetch('/user/all', {
                    method: 'GET',
                    credentials: 'include'
                })
                .then(response => response.json())
                .then(users => {
                    totalUsersCount = users.length;
                    // Теперь загружаем пользователей с пагинацией
                    loadUsersData();
                });
            }
        });
    });
    
    // Переключение между вкладками в разделе ролей
    tabBtns.forEach(btn => {
        btn.addEventListener('click', function() {
            const tabId = this.getAttribute('data-tab');
            
            // Убираем активный класс у всех кнопок и контента
            tabBtns.forEach(b => b.classList.remove('active'));
            tabContents.forEach(c => c.classList.remove('active'));
            
            // Добавляем активный класс текущей кнопке и контенту
            this.classList.add('active');
            document.getElementById(`${tabId}-tab`).classList.add('active');
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
            case 'roles':
                loadRolesData();
                break;
            case 'currency':
                loadCurrencyData();
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
            
            // Получаем ID текущего пользователя для сравнения
            return fetch('/user/profile', {
                method: 'GET',
                credentials: 'include'
            })
            .then(response => {
                if (!response.ok) throw new Error('Ошибка загрузки профиля');
                return response.json();
            })
            .then(currentUser => {
                return { users, currentUserId: currentUser.id };
            });
        })
        .then(({ users, currentUserId }) => {
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
                    return { ...user, roles, relationTypes, isCurrentUser: user.id === currentUserId };
                } catch (error) {
                    console.error(`Ошибка загрузки ролей для пользователя ${user.id}:`, error);
                    return { ...user, roles: ['Ошибка загрузки ролей'], relationTypes: [], isCurrentUser: user.id === currentUserId };
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
                    // Для клиента добавляем обе кнопки
                    const accountsBtn = document.createElement('button');
                    accountsBtn.className = 'action-btn accounts-user';
                    accountsBtn.dataset.userId = user.id;
                    accountsBtn.textContent = 'Все счета';
                    actionsTd.appendChild(accountsBtn);
                    
                    const editBtn = document.createElement('button');
                    editBtn.className = 'action-btn edit-user';
                    editBtn.dataset.userId = user.id;
                    editBtn.textContent = 'Редактировать';
                    // Добавляем обработчик клика
                    editBtn.addEventListener('click', () => showEditUserModal(user));
                    actionsTd.appendChild(editBtn);
                } else if (isSupport) {
                    // Для поддержки только кнопку редактирования
                    const editBtn = document.createElement('button');
                    editBtn.className = 'action-btn edit-user';
                    editBtn.dataset.userId = user.id;
                    editBtn.textContent = 'Редактировать';
                    // Добавляем обработчик клика
                    editBtn.addEventListener('click', () => showEditUserModal(user));
                    actionsTd.appendChild(editBtn);
                }
                
                // Добавляем кнопку "Сессии" для всех пользователей, кроме текущего
                if (!user.isCurrentUser) {
                    const sessionsBtn = document.createElement('button');
                    sessionsBtn.className = 'action-btn sessions-user';
                    sessionsBtn.dataset.userId = user.id;
                    sessionsBtn.textContent = 'Сессии';
                    actionsTd.appendChild(sessionsBtn);
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

    function handleNameChange(userId, modal) {
        return function() {
            const nameInput = document.getElementById('edit-user-name');
            const newName = nameInput.value.trim();
            
            if (!newName) {
                alert('Пожалуйста, введите новое имя');
                return;
            }
            
            const url = `/user/profile/changename?userID=${userId}`;
            const params = new URLSearchParams();
            params.append('name', newName);
            
            fetch(url, {
                method: 'PATCH',
                credentials: 'include',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: params
            })
            .then(response => {
                if (!response.ok) throw new Error('Ошибка при изменении имени');
                return response.json();
            })
            .then(data => {
                alert('Имя успешно изменено');
                loadUsersData();
                modal.classList.remove('active');
            })
            .catch(error => {
                console.error('Ошибка:', error);
                alert('Произошла ошибка при изменении имени: ' + error.message);
            });
        };
    }

    function handleEmailChange(userId, modal) {
        return function() {
            const emailInput = document.getElementById('edit-user-email');
            const newEmail = emailInput.value.trim();
            
            if (!newEmail) {
                alert('Пожалуйста, введите новый email');
                return;
            }
            
            if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(newEmail)) {
                alert('Пожалуйста, введите корректный email');
                return;
            }
            
            const url = `/user/profile/changeemail?userID=${userId}`;
            const params = new URLSearchParams();
            params.append('email', newEmail);
            
            fetch(url, {
                method: 'PATCH',
                credentials: 'include',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: params
            })
            .then(response => {
                if (!response.ok) throw new Error('Ошибка при изменении email');
                return response.json();
            })
            .then(data => {
                alert('Email успешно изменён');
                loadUsersData();
                modal.classList.remove('active');
            })
            .catch(error => {
                console.error('Ошибка:', error);
                alert('Произошла ошибка при изменении email: ' + error.message);
            });
        };
    }

    // Глобальные переменные для хранения текущих обработчиков
    let currentNameHandler = null;
    let currentEmailHandler = null;
    
    // Функция для показа модального окна редактирования пользователя
    function showEditUserModal(user) {
        const modal = document.getElementById('edit-user-modal');
        const title = document.getElementById('edit-user-title');
        const nameInput = document.getElementById('edit-user-name');
        const emailInput = document.getElementById('edit-user-email');
        const changeName = document.getElementById('change-name-btn');
        const changeEmail =  document.getElementById('change-email-btn');
        
        // Удаляем предыдущие обработчики, если они есть
        if (currentNameHandler) {
            changeName.removeEventListener('click', currentNameHandler);
        }
        if (currentEmailHandler) {
            changeEmail.removeEventListener('click', currentEmailHandler);
        }
        
        // Создаем новые обработчики
        currentNameHandler = handleNameChange(user.id, modal);
        currentEmailHandler = handleEmailChange(user.id, modal);
        
        // Добавляем новые обработчики
        changeName.addEventListener('click', currentNameHandler);
        changeEmail.addEventListener('click', currentEmailHandler);
        
        // Заполняем данные
        title.textContent = `Редактирование пользователя #${user.id}`;
        nameInput.value = user.full_name || '';
        emailInput.value = user.email || '';
        
        // Показываем модальное окно
        modal.classList.add('active');

        // Обработчик закрытия модального окна
        const closeBtn = modal.querySelector('.close-btn');
        closeBtn.addEventListener('click', () => {
            modal.classList.remove('active');
        });
        
        // Закрытие при клике вне модального окна
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.classList.remove('active');
            }
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
    
    // Глобальная переменная для хранения текущего ID платежа
    let currentTransactionId = null;

    // Функция для показа деталей платежа
    function showTransactionDetails(transactionId) {
        const modal = document.getElementById('transaction-details-modal');
        const content = document.getElementById('transaction-details-content');
        const actionsContainer = document.getElementById('transaction-actions');
        const updateForm = document.getElementById('update-status-form');
        
        // Очищаем предыдущие данные
        currentTransactionId = transactionId;
        content.innerHTML = '<div class="loading">Загрузка деталей платежа...</div>';
        actionsContainer.innerHTML = '';
        updateForm.style.display = 'none';
        
        // Показываем модальное окно
        modal.classList.add('active');
        
        // Загружаем детали платежа
        fetch(`/transactions/${transactionId}`, {
            method: 'GET',
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка загрузки деталей платежа');
            return response.json();
        })
        .then(transaction => {
            // Отображаем детали платежа
            content.innerHTML = `
                <div class="transaction-detail">
                    <span class="transaction-detail-label">ID:</span>
                    <span class="transaction-detail-value">${transaction.id}</span>
                </div>
                <div class="transaction-detail">
                    <span class="transaction-detail-label">Отправитель:</span>
                    <span class="transaction-detail-value">${transaction.sender}</span>
                </div>
                <div class="transaction-detail">
                    <span class="transaction-detail-label">Получатель:</span>
                    <span class="transaction-detail-value">${transaction.receiver}</span>
                </div>
                <div class="transaction-detail">
                    <span class="transaction-detail-label">Сумма:</span>
                    <span class="transaction-detail-value">${transaction.amount.toFixed(2)} ${transaction.currency}</span>
                </div>
                <div class="transaction-detail">
                    <span class="transaction-detail-label">Статус:</span>
                    <span class="transaction-detail-value status-${transaction.status.toLowerCase()}">${transaction.status}</span>
                </div>
                <div class="transaction-detail">
                    <span class="transaction-detail-label">Описание:</span>
                    <span class="transaction-detail-value">${transaction.description || 'Нет описания'}</span>
                </div>
                <div class="transaction-detail">
                    <span class="transaction-detail-label">Создан:</span>
                    <span class="transaction-detail-value">${new Date(transaction.createdAt).toLocaleString()}</span>
                </div>
                <div class="transaction-detail">
                    <span class="transaction-detail-label">Обновлен:</span>
                    <span class="transaction-detail-value">${transaction.updatedAt ? new Date(transaction.updatedAt).toLocaleString() : 'Не обновлялся'}</span>
                </div>
            `;
            
            // Добавляем кнопки действий в зависимости от статуса
            if (transaction.status.toLowerCase() !== 'cancelled') {
                if (transaction.status.toLowerCase() !== 'completed') {
                    const cancelBtn = document.createElement('button');
                    cancelBtn.className = 'action-btn danger';
                    cancelBtn.textContent = 'Отменить';
                    cancelBtn.onclick = () => cancelTransaction(transactionId);
                    actionsContainer.appendChild(cancelBtn);
                }
                
                if (transaction.status.toLowerCase() !== 'completed') {
                    const updateBtn = document.createElement('button');
                    updateBtn.className = 'action-btn';
                    updateBtn.textContent = 'Обновить статус';
                    updateBtn.onclick = () => {
                        updateForm.style.display = updateForm.style.display === 'none' ? 'block' : 'none';
                    };
                    actionsContainer.appendChild(updateBtn);
                    
                    // Обработчик для кнопки обновления статуса
                    document.getElementById('update-status-btn').onclick = () => {
                        const status = document.getElementById('status-select').value;
                        updateTransactionStatus(transactionId, status);
                    };
                }
            }
        })
        .catch(error => {
            console.error('Ошибка загрузки деталей платежа:', error);
            content.innerHTML = `<div class="error">${error.message}</div>`;
        });
        
        // Обработчик закрытия модального окна
        const closeBtn = modal.querySelector('.close-btn');
        closeBtn.onclick = () => {
            modal.classList.remove('active');
        };
        
        // Закрытие при клике вне модального окна
        modal.onclick = (e) => {
            if (e.target === modal) {
                modal.classList.remove('active');
            }
        };
    }

    // Функция для отмены платежа
    function cancelTransaction(transactionId) {
        if (!confirm('Вы уверены, что хотите отменить этот платеж?')) return;
        
        fetch(`/transactions/${transactionId}/cancel`, {
            method: 'PATCH',
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка отмены платежа');
            return response.text(); // Может быть пустой ответ
        })
        .then(() => {
            alert('Платеж успешно отменен');
            // Перезагружаем детали платежа
            showTransactionDetails(transactionId);
        })
        .catch(error => {
            console.error('Ошибка отмены платежа:', error);
            alert(error.message);
        });
    }

    // Функция для обновления статуса платежа
    function updateTransactionStatus(transactionId, status) {
        const formData = new URLSearchParams();
        formData.append('status', status);
        
        fetch(`/transactions/${transactionId}/update`, {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: formData,
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка обновления статуса платежа');
            return response.text(); // Может быть пустой ответ
        })
        .then(() => {
            alert('Статус платежа успешно обновлен');
            // Перезагружаем детали платежа
            showTransactionDetails(transactionId);
            // Скрываем форму обновления
            document.getElementById('update-status-form').style.display = 'none';
        })
        .catch(error => {
            console.error('Ошибка обновления статуса платежа:', error);
            alert(error.message);
        });
    }

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
                
                // Добавляем кнопку "Детали платежа" только если есть TransactionID
                const transactionDetailsBtn = operation.transaction_id ? 
                    `<button class="action-btn view-transaction-details" data-transaction-id="${operation.transaction_id}">Детали платежа</button>` : 
                    '';
                
                html += `
                    <div class="operation-item">
                        <div class="operation-amount ${amountClass}">${amountSign}${operation.amount.toFixed(2)} ${operation.currency}</div>
                        <div class="operation-details">
                            <span class="operation-type">${operation.operation_type}</span>
                            ${operation.description ? `<div>${operation.description}</div>` : ''}
                        </div>
                        <div class="operation-date">${new Date(operation.created_at).toLocaleString()} (обработано: ${processedAt})</div>
                        <div class="operation-actions">
                            ${transactionDetailsBtn}
                        </div>
                    </div>
                `;
            });
            
            operationsContainer.innerHTML = html;
            
            // Добавляем обработчики для кнопок "Детали платежа"
            document.querySelectorAll('.view-transaction-details').forEach(btn => {
                btn.onclick = function() {
                    const transactionId = this.dataset.transactionId;
                    showTransactionDetails(transactionId);
                };
            });
            
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

    // Функция для показа модального окна сессий
    function showUserSessionsModal(userId) {
        const modal = document.getElementById('sessions-modal');
        const title = document.getElementById('sessions-modal-title');
        const sessionsList = document.getElementById('sessions-list');
        const logoutAllBtn = document.getElementById('logout-all-btn');

        const closeBtn = modal.querySelector('.close-btn');
        closeBtn.onclick = null;
        modal.onclick = null;
        logoutAllBtn.onclick = null;

        // Затем показываем модальное окно
        modal.classList.add('active');
        
        // Устанавливаем заголовок и загружаем данные
        title.textContent = `Сессии пользователя #${userId}`;
        sessionsList.innerHTML = '<div class="loading">Загрузка сессий...</div>';
        
        // Сохраняем ID пользователя
        currentSessionsUserId = userId;

        // Загружаем сессии пользователя
        loadUserSessions(userId);
        

        // Показываем модальное окно
        modal.classList.add('active');
        
        // Обработчик закрытия модального окна
        closeBtn.onclick = () => {
            modal.classList.remove('active');
        };
        
        // Закрытие при клике вне модального окна
        modal.onclick = (e) => {
            if (e.target === modal) {
                modal.classList.remove('active');
            }
        };
        
        // Обработчик кнопки "Завершить все сессии"
        logoutAllBtn.onclick = () => {
            if (confirm('Вы уверены, что хотите завершить все сессии этого пользователя?')) {
                logoutAllSessions(userId);
            }
        };
    }

    // Функция загрузки сессий пользователя
    function loadUserSessions(userId) {
        const sessionsList = document.getElementById('sessions-list');
        
        fetch(`/auth/sessions?userID=${userId}`, {
            method: 'GET',
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка загрузки сессий');
            return response.json();
        })
        .then(sessions => {
            sessionsList.innerHTML = '';
            
            if (!sessions || sessions.length === 0) {
                sessionsList.innerHTML = '<div class="empty">Нет активных сессий</div>';
                return;
            }
            
            sessions.forEach(session => {
                const sessionItem = document.createElement('div');
                sessionItem.className = 'session-item';
                
                const isExpired = new Date(session.expires_at) < new Date();
                
                sessionItem.innerHTML = `
                    <div class="session-info">
                        <div class="session-ip">${session.user_ip || 'IP не указан'}</div>
                        <div class="session-user-agent">${session.user_agent || 'User Agent не указан'}</div>
                        <div class="session-date">Создана: ${new Date(session.created_at).toLocaleString()}</div>
                        <div class="session-expires ${isExpired ? 'expired' : 'active'}">
                            ${isExpired ? 'Истекла' : 'Действует до'}: ${new Date(session.expires_at).toLocaleString()}
                        </div>
                    </div>
                    <div class="session-actions">
                        <button class="action-btn view-session-details" data-session-id="${session.id}">Детали</button>
                        <button class="action-btn danger logout-session" data-session-id="${session.id}">Завершить</button>
                    </div>
                `;
                
                sessionsList.appendChild(sessionItem);
            });
            
            // Добавляем обработчики для кнопок в сессиях
            document.querySelectorAll('.view-session-details').forEach(btn => {
                btn.addEventListener('click', function() {
                    const sessionId = this.dataset.sessionId;
                    showSessionDetails(sessionId, sessions);
                });
            });
            
            document.querySelectorAll('.logout-session').forEach(btn => {
                btn.addEventListener('click', function() {
                    const sessionId = this.dataset.sessionId;
                    if (confirm('Вы уверены, что хотите завершить эту сессию?')) {
                        logoutSession(sessionId, sessions);
                    }
                });
            });
        })
        .catch(error => {
            console.error('Ошибка загрузки сессий:', error);
            sessionsList.innerHTML = `<div class="error">${error.message}</div>`;
        });
    }

    // Функция для завершения всех сессий пользователя
    function logoutAllSessions(userId) {
        const logoutAllBtn = document.getElementById('logout-all-btn');
        const originalText = logoutAllBtn.textContent;
        
        logoutAllBtn.disabled = true;
        logoutAllBtn.textContent = 'Завершить все сессии';
        
        fetch(`/auth/logout-all?userID=${userId}`, {
            method: 'POST',
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка завершения всех сессий');
            return response.json();
        })
        .then(() => {
            alert('Все сессии пользователя успешно завершены');
            loadUserSessions(userId);
        })
        .catch(error => {
            console.error('Ошибка завершения всех сессий:', error);
            alert(error.message);
        })
        .finally(() => {
            logoutAllBtn.disabled = false;
            logoutAllBtn.textContent = originalText;
        });
    }

    // Функция для завершения конкретной сессии
    function logoutSession(sessionId, sessions) {
         // Находим сессию по ID
        const session = sessions.find(s => s.id === sessionId);
        if (!session) {
            alert('Сессия не найдена');
            return;
        }
        fetch(`/auth/logout?userID=${session.user_id}&sessionID=${session.id}`, {
            method: 'POST',
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка завершения сессии');
            return response.json();
        })
        .then(() => {
            alert('Сессия успешно завершена');
            loadUserSessions(session.user_id);
        })
        .catch(error => {
            console.error('Ошибка завершения сессии:', error);
            alert(error.message);
        });
    }

    // Функция для показа деталей сессии
    function showSessionDetails(sessionId, sessions) {
        const modal = document.getElementById('session-details-modal');
        const content = document.getElementById('session-details-content');
        
        // Находим сессию по ID
        const session = sessions.find(s => s.id === sessionId);
        if (!session) {
            alert('Сессия не найдена');
            return;
        }
        
        const isExpired = new Date(session.expires_at) < new Date();
        
        content.innerHTML = `
            <div class="session-detail">
                <span class="session-detail-label">ID сессии:</span>
                <span class="session-detail-value">${session.id}</span>
            </div>
            <div class="session-detail">
                <span class="session-detail-label">ID пользователя:</span>
                <span class="session-detail-value">${session.user_id}</span>
            </div>
            <div class="session-detail">
                <span class="session-detail-label">IP адрес:</span>
                <span class="session-detail-value">${session.user_ip || 'Не указан'}</span>
            </div>
            <div class="session-detail">
                <span class="session-detail-label">User Agent:</span>
                <span class="session-detail-value">${session.user_agent || 'Не указан'}</span>
            </div>
            <div class="session-detail">
                <span class="session-detail-label">Создана:</span>
                <span class="session-detail-value">${new Date(session.created_at).toLocaleString()}</span>
            </div>
            <div class="session-detail">
                <span class="session-detail-label">Истекает:</span>
                <span class="session-detail-value ${isExpired ? 'expired' : 'active'}">
                    ${new Date(session.expires_at).toLocaleString()} (${isExpired ? 'истекла' : 'активна'})
                </span>
            </div>
        `;
        
        modal.classList.add('active');
        
        // Обработчик закрытия модального окна
        const closeBtn = modal.querySelector('.close-btn');
        closeBtn.addEventListener('click', () => {
            modal.classList.remove('active');
        });
        
        // Закрытие при клике вне модального окна
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.classList.remove('active');
            }
        });
    }

    // Добавляем обработчик для кнопок "Сессии" в таблице пользователей
    document.addEventListener('click', function(e) {
        if (e.target && e.target.classList.contains('sessions-user')) {
            const userId = e.target.dataset.userId;
            showUserSessionsModal(userId);
        }
    });

    // Глобальные переменные для хранения обработчиков
    const rolesHandlers = {
        createRelation: null,
        assignPermission: null,
        revokePermission: null,
        loadUserData: null,
        relationsContainerClick: null
    };

    // Функция очистки обработчиков
    function cleanupRolesHandlers() {
        const createRelationBtn = document.getElementById('create-relation-btn');
        const assignPermissionBtn = document.getElementById('assign-permission-btn');
        const revokePermissionBtn = document.getElementById('revoke-permission-btn');
        const loadUserDataBtn = document.getElementById('load-user-data-btn');
        const relationsContainer = document.getElementById('relations-container');
        
        if (rolesHandlers.createRelation && createRelationBtn) {
            createRelationBtn.removeEventListener('click', rolesHandlers.createRelation);
        }
        if (rolesHandlers.assignPermission && assignPermissionBtn) {
            assignPermissionBtn.removeEventListener('click', rolesHandlers.assignPermission);
        }
        if (rolesHandlers.revokePermission && revokePermissionBtn) {
            revokePermissionBtn.removeEventListener('click', rolesHandlers.revokePermission);
        }
        if (rolesHandlers.loadUserData && loadUserDataBtn) {
            loadUserDataBtn.removeEventListener('click', rolesHandlers.loadUserData);
        }
        if (rolesHandlers.relationsContainerClick && relationsContainer) {
            relationsContainer.removeEventListener('click', rolesHandlers.relationsContainerClick);
        }
    }

    // Функция для создания связи
    function createRelation(sourceId, targetId, relationType) {
        return fetch('/role/relations', {
            method: 'POST',
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                sourceID: sourceId,
                targetID: targetId,
                relationType: relationType
            })
        });
    }

    // Функция удаления связи
    function deleteRelation(sourceId, targetId, relationType) {
        return fetch('/role/relations', {
            method: 'DELETE',
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                sourceID: sourceId,
                targetID: targetId,
                relationType: relationType
            })
        });
    }

    // Функция для назначения разрешения
    function assignPermission(permissionId, relationType) {
        return fetch('/role/permissions/assign', {
            method: 'POST',
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                permissionID: permissionId,
                relationType: relationType
            })
        });
    }

    // Функция для отзыва разрешения
    function revokePermission(permissionId, relationType) {
        return fetch('/role/permissions/revoke', {
            method: 'DELETE',
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                permissionID: permissionId,
                relationType: relationType
            })
        });
    }

    // Загрузка данных для раздела ролей
    function loadRolesData() {
        // Очищаем предыдущие обработчики
        cleanupRolesHandlers();
        
        // Получаем элементы DOM
        const userIdInput = document.getElementById('user-id-input');
        const relationsContainer = document.getElementById('relations-container');
        const permissionsContainer = document.getElementById('permissions-container');
        const createRelationBtn = document.getElementById('create-relation-btn');
        const assignPermissionBtn = document.getElementById('assign-permission-btn');
        const revokePermissionBtn = document.getElementById('revoke-permission-btn');
        const loadUserDataBtn = document.getElementById('load-user-data-btn');
        
        // Создаем и сохраняем новые обработчики
        rolesHandlers.loadUserData = function() {
            const userId = userIdInput.value.trim();
            if (!userId) {
                alert('Пожалуйста, введите ID пользователя');
                return;
            }
            loadUserRelations(userId);
            loadUserPermissions(userId);
        };
        
        rolesHandlers.createRelation = function() { showCreateRelationModal(); };
        rolesHandlers.assignPermission = function() { showAssignPermissionModal(); };
        rolesHandlers.revokePermission = function() { showRevokePermissionModal(); };

         // Обработчик для делегирования событий в контейнере связей
        rolesHandlers.relationsContainerClick = function(e) {
            if (e.target.classList.contains('info-btn')) {
                const relationType = e.target.dataset.relationType;
                showRelationPermissionsModal(relationType);
            }
            
            if (e.target.classList.contains('delete-btn')) {
                const relationType = e.target.dataset.relationType;
                const source = e.target.dataset.sourceId;
                const target = e.target.dataset.targetId;
                deleteRelation(source, target, relationType)
                .then(async response => {
                    if (!response.ok) throw new Error('Ошибка удаления связи');
                    const text = await response.text();
                    return text ? JSON.parse(text) : {};
                })
                .then(() => {
                    alert('Связь успешно удалена');
                    const userId = userIdInput.value.trim();
                    if (userId) loadUserRelations(userId);
                })
                .catch(error => {
                    console.error('Ошибка удаления связи:', error);
                    alert(error.message);
                });
            }
        };

        // Добавляем обработчики
        if (loadUserDataBtn) loadUserDataBtn.addEventListener('click', rolesHandlers.loadUserData);
        if (createRelationBtn) createRelationBtn.addEventListener('click', rolesHandlers.createRelation);
        if (assignPermissionBtn) assignPermissionBtn.addEventListener('click', rolesHandlers.assignPermission);
        if (revokePermissionBtn) revokePermissionBtn.addEventListener('click', rolesHandlers.revokePermission);
        if (relationsContainer) relationsContainer.addEventListener('click', rolesHandlers.relationsContainerClick);
            
        // Функция загрузки связей пользователя
        function loadUserRelations(userId) {
            fetch(`/role/users/${userId}/relations`, {
                method: 'GET',
                credentials: 'include',
                headers: {
                    'Accept': 'application/json'
                }
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                return response.json();
            })
            .then(relations => {
                displayUserRelations(relations);
            })
            .catch(error => {
                console.error('Ошибка загрузки связей:', error);
                if (relationsContainer) {
                    relationsContainer.innerHTML = `<div class="error">Ошибка загрузки связей: ${error.message}</div>`;
                }
            });
        }
        
        // Функция отображения связей пользователя
        function displayUserRelations(relations) {
            if (!relationsContainer) return;
            
            relationsContainer.innerHTML = '';
            
            if (!relations || relations.length === 0) {
                relationsContainer.innerHTML = '<div class="empty">Нет связей</div>';
                return;
            }
            
            const relationsList = document.createElement('div');
            relationsList.className = 'relations-list';
            
            relations.forEach(relation => {
                const relationItem = document.createElement('div');
                relationItem.className = 'relation-item';
                
                // Получаем информацию о сущностях
                Promise.all([
                    fetchEntityInfo(relation.SourceID),
                    fetchEntityInfo(relation.TargetID)
                ])
                .then(([source, target]) => {
                    relationItem.innerHTML = `
                        <div class="relation-info">
                            <strong>${source?.Type || relation.SourceID}</strong> → 
                            <strong>${target?.Type || relation.TargetID}</strong>
                            <span class="relation-type">(${relation.RelationType})</span>
                        </div>
                        <div class="relation-actions">
                            <button class="info-btn" data-relation-type="${relation.RelationType}">Информация</button>
                            <button class="delete-btn" 
                                data-relation-type="${relation.RelationType}"
                                data-source-id="${relation.SourceID}"
                                data-target-id="${relation.TargetID}">
                                Удалить
                            </button>
                        </div>
                    `;
                })
                .catch(error => {
                    console.error('Ошибка загрузки информации о сущностях:', error);
                    relationItem.innerHTML = `
                        <div class="relation-info">
                            <strong>${relation.SourceID}</strong> → 
                            <strong>${relation.TargetID}</strong>
                            <span class="relation-type">(${relation.RelationType})</span>
                        </div>
                        <div class="relation-actions">
                            <button class="info-btn" data-relation-type="${relation.RelationType}">Информация</button>
                            <button class="delete-btn" 
                                data-relation-type="${relation.RelationType}"
                                data-source-id="${relation.SourceID}"
                                data-target-id="${relation.TargetID}">
                                Удалить
                            </button>
                        </div>
                    `;
                });
                
                relationsList.appendChild(relationItem);
            });
            
            relationsContainer.appendChild(relationsList);
        }
        
        // Функция загрузки информации о сущности
        async function fetchEntityInfo(entityId) {
            const response = await fetch('/role/entities', {
                method: 'GET',
                credentials: 'include',
                headers: {
                    'Accept': 'application/json'
                }
            });
            if (!response.ok) throw new Error('Failed to fetch entities');
            const entities = await response.json();
            return entities.find(entity => entity.ID === entityId) || null;
        }
        
        // Функция загрузки разрешений пользователя
        function loadUserPermissions(userId) {
            fetch(`/role/users/${userId}/permissions`, {
                method: 'GET',
                credentials: 'include',
                headers: {
                    'Accept': 'application/json'
                }
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                return response.json();
            })
            .then(permissions => {
                displayUserPermissions(permissions);
            })
            .catch(error => {
                console.error('Ошибка загрузки разрешений:', error);
                if (permissionsContainer) {
                    permissionsContainer.innerHTML = `<div class="error">Ошибка загрузки разрешений: ${error.message}</div>`;
                }
            });
        }
        
        // Функция отображения разрешений пользователя
        function displayUserPermissions(permissions) {
            if (!permissionsContainer) return;
            
            permissionsContainer.innerHTML = '';
            
            if (!permissions || permissions.length === 0) {
                permissionsContainer.innerHTML = '<div class="empty">Нет разрешений</div>';
                return;
            }
            
            const permissionsList = document.createElement('div');
            permissionsList.className = 'permissions-list';
            
            permissions.forEach(permission => {
                const permissionItem = document.createElement('div');
                permissionItem.className = 'permission-item';
                permissionItem.innerHTML = `
                    <div class="permission-name">${permission.Name}</div>
                    <div class="permission-description">${permission.Description || 'Нет описания'}</div>
                `;
                permissionsList.appendChild(permissionItem);
            });
            
            permissionsContainer.appendChild(permissionsList);
        }
        
        // Модальное окно для создания связи
        function showCreateRelationModal() {
            const modal = document.createElement('div');
            modal.className = 'modal';
            modal.innerHTML = `
                <div class="modal-content">
                    <span class="close-btn">&times;</span>
                    <h2>Создать связь</h2>
                    <form id="create-relation-modal-form">
                        <div class="form-group">
                            <label for="relation-type-input">Тип связи</label>
                            <input type="text" id="relation-type-input" required>
                        </div>
                        <div class="form-group">
                            <label for="relation-source-select">Ресурс</label>
                            <select id="relation-source-select" required>
                                <option value="">Выберите ресурс</option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label for="relation-target-select">Цель</label>
                            <select id="relation-target-select" required>
                                <option value="">Выберите цель</option>
                            </select>
                        </div>
                        <button type="submit">Создать</button>
                    </form>
                </div>
            `;
            
            document.body.appendChild(modal);
            
            // Загружаем сущности для выпадающих списков
            fetch('/role/entities', {
                method: 'GET',
                credentials: 'include',
                headers: {
                    'Accept': 'application/json'
                }
            })
            .then(response => {
                if (!response.ok) throw new Error('Failed to fetch entities');
                return response.json();
            })
            .then(entities => {
                const sourceSelect = modal.querySelector('#relation-source-select');
                const targetSelect = modal.querySelector('#relation-target-select');
                
                // Добавляем текущего пользователя как возможную сущность
                const userId = userIdInput.value.trim();
                if (userId) {
                    const userOption = document.createElement('option');
                    userOption.value = userId;
                    userOption.textContent = 'Пользователь';
                    sourceSelect.appendChild(userOption.cloneNode(true));
                    targetSelect.appendChild(userOption.cloneNode(true));
                }
                
                // Добавляем остальные сущности
                entities.forEach(entity => {
                    const option = document.createElement('option');
                    option.value = entity.ID;
                    option.textContent = entity.Type + entity.ID;
                    sourceSelect.appendChild(option.cloneNode(true));
                    targetSelect.appendChild(option.cloneNode(true));
                });
            })
            .catch(error => {
                console.error('Ошибка загрузки сущностей:', error);
            });
            
            // Обработчик закрытия модального окна
            modal.querySelector('.close-btn').addEventListener('click', () => {
                modal.remove();
            });
            
            // Обработчик клика вне модального окна
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    modal.remove();
                }
            });
            
            // Обработчик формы создания связи
            modal.querySelector('#create-relation-modal-form').addEventListener('submit', function(e) {
                e.preventDefault();
                
                const relationType = modal.querySelector('#relation-type-input').value;
                const source = modal.querySelector('#relation-source-select').value;
                const target = modal.querySelector('#relation-target-select').value;
                
                createRelation(source, target, relationType)
                .then(async response => {
                    if (!response.ok) throw new Error('Ошибка создания связи');
                    const text = await response.text();
                    return text ? JSON.parse(text) : {};
                })
                .then(() => {
                    alert('Связь успешно создана');
                    modal.remove();
                    const userId = userIdInput.value.trim();
                    if (userId) loadUserRelations(userId);
                })
                .catch(error => {
                    console.error('Ошибка создания связи:', error);
                    alert(error.message);
                });
            });
        }
        
        // Модальное окно для назначения разрешения
        function showAssignPermissionModal() {
            const modal = document.createElement('div');
            modal.className = 'modal';
            modal.innerHTML = `
                <div class="modal-content">
                    <span class="close-btn">&times;</span>
                    <h2>Назначить разрешение</h2>
                    <form id="assign-permission-modal-form">
                        <div class="form-group">
                            <label for="assign-relation-type-input">Тип связи</label>
                            <input type="text" id="assign-relation-type-input" required>
                        </div>
                        <div class="form-group">
                            <label for="assign-permission-select">Разрешение</label>
                            <select id="assign-permission-select" required>
                                <option value="">Выберите разрешение</option>
                            </select>
                        </div>
                        <div class="warning">
                            Внимание: Добавление разрешения для выбранной связи приведёт к изменению прав всех пользователей, имеющих этот тип связи
                        </div>
                        <button type="submit">Назначить</button>
                    </form>
                </div>
            `;
            
            document.body.appendChild(modal);
            
            // Загружаем разрешения для выпадающего списка
            fetch('/role/permissions', {
                method: 'GET',
                credentials: 'include',
                headers: {
                    'Accept': 'application/json'
                }
            })
            .then(response => {
                if (!response.ok) throw new Error('Failed to fetch permissions');
                return response.json();
            })
            .then(permissions => {
                const permissionSelect = modal.querySelector('#assign-permission-select');
                if (!permissionSelect) return;
                
                permissions.forEach(permission => {
                    const option = document.createElement('option');
                    option.value = permission.ID;
                    option.textContent = permission.Name;
                    permissionSelect.appendChild(option);
                });
            })
            .catch(error => {
                console.error('Ошибка загрузки разрешений:', error);
            });
            
            // Обработчик закрытия модального окна
            modal.querySelector('.close-btn').addEventListener('click', () => {
                modal.remove();
            });
            
            // Обработчик клика вне модального окна
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    modal.remove();
                }
            });
            
            // Обработчик формы назначения разрешения
            modal.querySelector('#assign-permission-modal-form').addEventListener('submit', function(e) {
                e.preventDefault();
                
                const relationType = modal.querySelector('#assign-relation-type-input').value;
                const permissionId = modal.querySelector('#assign-permission-select').value;
                
                assignPermission(permissionId, relationType)
                .then(async response => {
                    if (!response.ok) throw new Error('Ошибка назначения разрешения');
                    const text = await response.text();
                    return text ? JSON.parse(text) : {};
                })
                .then(() => {
                    alert('Разрешение успешно назначено');
                    modal.remove();
                    const userId = userIdInput.value.trim();
                    if (userId) loadUserPermissions(userId);
                })
                .catch(error => {
                    console.error('Ошибка назначения разрешения:', error);
                    alert(error.message);
                });
            });
        }
        
        // Модальное окно для отзыва разрешения
        function showRevokePermissionModal() {
            const modal = document.createElement('div');
            modal.className = 'modal';
            modal.innerHTML = `
                <div class="modal-content">
                    <span class="close-btn">&times;</span>
                    <h2>Отозвать разрешение</h2>
                    <form id="revoke-permission-modal-form">
                        <div class="form-group">
                            <label for="revoke-relation-type-input">Тип связи</label>
                            <input type="text" id="revoke-relation-type-input" required>
                        </div>
                        <div class="form-group">
                            <label for="revoke-permission-select">Разрешение</label>
                            <select id="revoke-permission-select" required>
                                <option value="">Выберите разрешение</option>
                            </select>
                        </div>
                        <div class="warning">
                            Внимание: Отзыв разрешения у выбранной связи приведёт к изменению прав всех пользователей, имеющих этот тип связи
                        </div>
                        <button type="submit">Отозвать</button>
                    </form>
                </div>
            `;
            
            document.body.appendChild(modal);
            
            // Загружаем разрешения для выпадающего списка
            fetch('/role/permissions', {
                method: 'GET',
                credentials: 'include',
                headers: {
                    'Accept': 'application/json'
                }
            })
            .then(response => {
                if (!response.ok) throw new Error('Failed to fetch permissions');
                return response.json();
            })
            .then(permissions => {
                const permissionSelect = modal.querySelector('#revoke-permission-select');
                if (!permissionSelect) return;
                
                permissions.forEach(permission => {
                    const option = document.createElement('option');
                    option.value = permission.ID;
                    option.textContent = permission.Name;
                    permissionSelect.appendChild(option);
                });
            })
            .catch(error => {
                console.error('Ошибка загрузки разрешений:', error);
            });
            
            // Обработчик закрытия модального окна
            modal.querySelector('.close-btn').addEventListener('click', () => {
                modal.remove();
            });
            
            // Обработчик клика вне модального окна
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    modal.remove();
                }
            });
            
            // Обработчик формы отзыва разрешения
            modal.querySelector('#revoke-permission-modal-form').addEventListener('submit', function(e) {
                e.preventDefault();
                
                const relationType = modal.querySelector('#revoke-relation-type-input').value;
                const permissionId = modal.querySelector('#revoke-permission-select').value;
                
                revokePermission(permissionId, relationType)
                .then(async response => {
                    if (!response.ok) throw new Error('Ошибка отзыва разрешения');
                    const text = await response.text();
                    return text ? JSON.parse(text) : {};
                })
                .then(() => {
                    alert('Разрешение успешно отозвано');
                    modal.remove();
                    const userId = userIdInput.value.trim();
                    if (userId) loadUserPermissions(userId);
                })
                .catch(error => {
                    console.error('Ошибка отзыва разрешения:', error);
                    alert(error.message);
                });
            });
        }
        
        // Модальное окно для просмотра разрешений типа связи
        function showRelationPermissionsModal(relationType) {
            const modal = document.createElement('div');
            modal.className = 'modal';
            modal.innerHTML = `
                <div class="modal-content">
                    <span class="close-btn">&times;</span>
                    <h2>Разрешения для типа связи: ${relationType}</h2>
                    <div class="relation-permissions-list" id="relation-permissions-list">
                        <div class="loading">Загрузка разрешений...</div>
                    </div>
                </div>
            `;
            
            document.body.appendChild(modal);
            
            // Загружаем разрешения для данного типа связи
            fetch(`/role/relations/${encodeURIComponent(relationType)}/permissions`, {
                method: 'GET',
                credentials: 'include',
                headers: {
                    'Accept': 'application/json'
                }
            })
            .then(response => {
                if (!response.ok) throw new Error('Failed to fetch relation permissions');
                return response.json();
            })
            .then(permissions => {
                const permissionsList = modal.querySelector('#relation-permissions-list');
                if (!permissionsList) return;
                
                permissionsList.innerHTML = '';
                
                if (!permissions || permissions.length === 0) {
                    permissionsList.innerHTML = '<div class="empty">Нет разрешений для этого типа связи</div>';
                    return;
                }
                
                permissions.forEach(permission => {
                    const permissionItem = document.createElement('div');
                    permissionItem.className = 'permission-item';
                    permissionItem.innerHTML = `
                        <div class="permission-name">${permission.Name}</div>
                        <div class="permission-description">${permission.Description || 'Нет описания'}</div>
                    `;
                    permissionsList.appendChild(permissionItem);
                });
            })
            .catch(error => {
                console.error('Ошибка загрузки разрешений типа связи:', error);
                const permissionsList = modal.querySelector('#relation-permissions-list');
                if (permissionsList) {
                    permissionsList.innerHTML = `<div class="error">Ошибка загрузки разрешений: ${error.message}</div>`;
                }
            });
            
            // Обработчик закрытия модального окна
            modal.querySelector('.close-btn').addEventListener('click', () => {
                modal.remove();
            });
            
            // Обработчик клика вне модального окна
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    modal.remove();
                }
            });
        }
    }
        
    function loadCurrencyData() {
        const currencyRates = document.getElementById('currency-rates');
        currencyRates.innerHTML = '<div class="loading">Загрузка курсов...</div>';
        
        const currencyPairs = [
            { from: 'USD', to: 'RUB' },
            { from: 'RUB', to: 'USD' },
            { from: 'EUR', to: 'RUB' },
            { from: 'RUB', to: 'EUR' },
            { from: 'USD', to: 'EUR' },
            { from: 'EUR', to: 'USD' }
        ];
        
        Promise.all(currencyPairs.map(pair => 
            fetch(`/transactions/currency-rate/get?from=${pair.from}&to=${pair.to}`, {
                method: 'GET',
                credentials: 'include'
            })
            .then(response => {
                if (!response.ok) throw new Error('Ошибка загрузки курса');
                return response.json();
            })
        ))
        .then(rates => {
            currencyRates.innerHTML = '';
            
            if (rates.length === 0) {
                currencyRates.innerHTML = '<div class="empty">Нет данных о курсах</div>';
                return;
            }
            
            rates.forEach(rate => {
                const item = document.createElement('div');
                item.className = 'currency-rate-item';
                item.innerHTML = `
                    <div>${rate.from_currency} → ${rate.to_currency}</div>
                    <div>${rate.rate.toFixed(4)}</div>
                `;
                currencyRates.appendChild(item);
            });
        })
        .catch(error => {
            console.error('Ошибка загрузки курсов:', error);
            currencyRates.innerHTML = `<div class="error">${error.message}</div>`;
        });
    }

    // Обработка формы обновления курса валют
    const currencyRateForm = document.getElementById('currency-rate-form');
    if (currencyRateForm) {
        currencyRateForm.addEventListener('submit', function(e) {
            e.preventDefault();
            
            const fromCurrency = document.getElementById('from-currency').value;
            const toCurrency = document.getElementById('to-currency').value;
            const rate = document.getElementById('rate').value;
            
            fetch('/transactions/currency-rate', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    from_currency: fromCurrency,
                    to_currency: toCurrency,
                    rate: parseFloat(rate)
                }),
                credentials: 'include'
            })
            .then(async response => {
                if (!response.ok) {
                    // Если статус ответа не 2xx, пробуем получить текст ошибки
                    const text = await response.text();
                    throw new Error(text || 'Ошибка обновления курса');
                }
                // Если ответ успешный и пустой, просто продолжаем
                return Promise.resolve();
            })
            .then(() => {
                alert('Курс обновлен успешно');
                loadCurrencyData(); // Обновляем список курсов
                currencyRateForm.reset();
            })
            .catch(error => {
                console.error('Ошибка обновления курса:', error);
                alert(error.message);
            });
        });
    }
    
    // Загрузка данных при первом открытии
    const activeSection = document.querySelector('.admin-section.active');
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
                //пропускаем
            } else if (role === 'support') {
                window.location.href = '/support';
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