document.addEventListener('DOMContentLoaded', function() {
document.getElementById('close-history').addEventListener('click', function() {
document.getElementById('history-modal').classList.add('hidden');
});
document.getElementById('prev-page').addEventListener('click', function() {
    if (currentPage > 1) {
        currentPage--;
        loadHistoryPage();
    }
});

document.getElementById('next-page').addEventListener('click', function() {
    currentPage++;
    loadHistoryPage();
});

    const accountsList = document.getElementById('accounts-list');
    const createAccountBtn = document.getElementById('create-account-btn');
    const createAccountModal = document.getElementById('create-account-modal');
    const closeModal = document.querySelector('.close');
    const createAccountForm = document.getElementById('create-account-form');
    
    // Загрузка счетов при открытии страницы
    loadAccounts();
    loadProfile();
    
    // Открытие модального окна для создания счета
    createAccountBtn.addEventListener('click', function() {
        createAccountModal.classList.remove('hidden');
    });
    
    // Закрытие модального окна
    closeModal.addEventListener('click', function() {
        createAccountModal.classList.add('hidden');
    });
    
    // Обработка создания нового счета
    createAccountForm.addEventListener('submit', function(e) {
        e.preventDefault();
        
        const accountName = document.getElementById('account-name').value;
        const accountCurrency = document.getElementById('account-currency').value;
        
        fetch('/account/create', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: `account_name=${encodeURIComponent(accountName)}&currency=${encodeURIComponent(accountCurrency)}`,
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка создания счета');
            return response.json();
        })
        .then(data => {
            createAccountModal.classList.add('hidden');
            createAccountForm.reset();
            loadAccounts(); // Обновляем список счетов
        })
        .catch(error => {
            alert(error.message);
        });
    });
    
    // Функция загрузки счетов
    function loadAccounts() {
        accountsList.innerHTML = '<div class="loading">Загрузка счетов...</div>';
        
        fetch('/account/get', {
            method: 'GET',
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка загрузки счетов');
            return response.json();
        })
        .then(accounts => {
            if (accounts.length === 0) {
                accountsList.innerHTML = '<div class="empty">У вас пока нет счетов</div>';
                return;
            }
            
            accountsList.innerHTML = '';
            accounts.forEach(account => {
                const accountCard = document.createElement('div');
                accountCard.className = 'account-card';
                accountCard.innerHTML = `
                    <h3>${account.account_name}</h3>
                    <div class="account-id">№ ${account.id}</div>
                    <div class="account-balance">${account.balance.toFixed(2)} ${account.currency}</div>
                    <div class="account-actions">
                        <button class="view-history" data-account-id="${account.id}">История операций</button>
                    </div>
                `;
                accountsList.appendChild(accountCard);
            });
            
            document.querySelectorAll('.view-history').forEach(btn => {
                btn.addEventListener('click', function() {
                    const accountId = this.getAttribute('data-account-id');
                    viewOperationHistory(accountId);
                });
            });
        })
        .catch(error => {
            accountsList.innerHTML = `<div class="error">${error.message}</div>`;
        });
    }
    
    let currentAccountId = null;
    let currentPage = 1;
    const limit = 10;

    function viewOperationHistory(accountId) {
        currentAccountId = accountId;
        currentPage = 1;
        loadHistoryPage();
        document.getElementById('history-modal').classList.remove('hidden');
    }

    function loadHistoryPage() {
    const offset = (currentPage - 1) * limit;
    const historyContainer = document.getElementById('history-container');
    historyContainer.innerHTML = '<div class="loading">Загрузка операций...</div>';
    
    fetch(`/transactions/history?account_id=${currentAccountId}&limit=${limit}&offset=${offset}`, {
        method: 'GET',
        credentials: 'include'
    })
    .then(response => {
        if (!response.ok) {
            // Добавим больше информации об ошибке
            return response.text().then(text => {
                throw new Error(text || 'Ошибка загрузки истории операций');
            });
        }
        return response.json();
    })
    .then(operations => {
        console.log('Received operations:', operations); // Логируем полученные данные
        historyContainer.innerHTML = '';
        
        if (!operations || operations.length === 0) {
            historyContainer.innerHTML = '<div class="empty">Нет операций на этой странице</div>';
            return;
        }
        
        operations.forEach(op => {
            const dateStr = op.created_at || op.processed_at;
            const timestamp = dateStr ? new Date(dateStr) : null;
            
            const formattedDate = timestamp 
                ? timestamp.toLocaleDateString('ru-RU') + ' ' + timestamp.toLocaleTimeString('ru-RU')
                : 'Дата не указана';
            
            const historyItem = document.createElement('div');
            historyItem.className = 'history-item';
            historyItem.innerHTML = `
                <div class="history-date">${formattedDate}</div>
                <div class="history-description">${op.description || 'Без описания'}</div>
                <div class="history-amount">${op.amount} ${op.currency}</div>
            `;
            historyContainer.appendChild(historyItem);
        });
        
        updatePaginationControls(operations.length);
    })
    .catch(error => {
        console.error('Error loading history:', error);
        historyContainer.innerHTML = `<div class="error">${error.message}</div>`;
    });
}

    function updatePaginationControls(itemsCount) {
        const prevBtn = document.getElementById('prev-page');
        const nextBtn = document.getElementById('next-page');
        const pageInfo = document.getElementById('page-info');
        
        pageInfo.textContent = `Страница ${currentPage}`;
        prevBtn.disabled = currentPage === 1;
        nextBtn.disabled = itemsCount < limit;
    }

    function updateAvatarLetter() {
        const username = document.getElementById('username').textContent;
        const avatar = document.getElementById('userAvatar');
        if (username && avatar) {
            avatar.textContent = username.charAt(0).toUpperCase();
        }
    }

    function loadProfile() {
        fetch('/user/profile', {
            method: 'GET',
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка загрузки профиля');
            return response.json();
        })
        .then(profile => {
            document.getElementById('username').textContent = profile.full_name;
            updateAvatarLetter();
        })
        .catch(error => {
            console.error('Error loading profile:', error);
        });
    }
});