document.addEventListener('DOMContentLoaded', function() {
    const tabBtns = document.querySelectorAll('.tab-btn');
    const tabContents = document.querySelectorAll('.tab-content');
    const transferForm = document.getElementById('transfer-form');
    const depositForm = document.getElementById('deposit-form');
    const convertForm = document.getElementById('convert-form');
    
    // Инициализация - показываем только первую вкладку
    document.querySelector('.tab-btn.active').classList.add('active');
    document.querySelector('.tab-content.active').classList.add('active');
    
    // Загрузка счетов при открытии страницы
    loadAccounts();
    loadProfile();

    // Переключение между вкладками
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
    
    // Обработка перевода
    transferForm.addEventListener('submit', function(e) {
        e.preventDefault();
        
        const fromAccount = document.getElementById('transfer-from').value;
        const toAccount = document.getElementById('transfer-to').value;
        const amount = document.getElementById('transfer-amount').value;
        const description = document.getElementById('transfer-description').value;
        
        fetch('/payments/transfer', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: `sender=${encodeURIComponent(fromAccount)}&receiver=${encodeURIComponent(toAccount)}&amount=${encodeURIComponent(amount)}&description=${encodeURIComponent(description)}`,
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка перевода');
            return response.json();
        })
        .then(data => {
            loadAccounts();
            alert('Перевод выполнен успешно');
            transferForm.reset();
        })
        .catch(error => {
            alert(error.message);
        });
    });
    
    // Обработка пополнения
    depositForm.addEventListener('submit', function(e) {
        e.preventDefault();
        
        const account = document.getElementById('deposit-account').value;
        const amount = document.getElementById('deposit-amount').value;
        const description = document.getElementById('deposit-description').value;
        
        fetch('/payments/deposit', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: `account_id=${encodeURIComponent(account)}&amount=${encodeURIComponent(amount)}&description=${encodeURIComponent(description)}`,
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка пополнения');
            return response.json();
        })
        .then(data => {
            loadAccounts();
            alert('Счёт успешно пополнен');
            depositForm.reset();
        })
        .catch(error => {
            alert(error.message);
        });
    });
    
    let currentFromCurrency = '';
    let currentToCurrency = '';
    let currentExchangeRate = 0;

    // обработчик формы конвертации
    convertForm.addEventListener('submit', function(e) {
        e.preventDefault();
        
        const fromAccount = document.getElementById('convert-from').value;
        const toAccount = document.getElementById('convert-to').value;
        const amount = document.getElementById('convert-amount').value;
        const description = document.getElementById('convert-description').value;
        
        fetch('/payments/convert', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: `sender=${encodeURIComponent(fromAccount)}&receiver=${encodeURIComponent(toAccount)}&amount=${encodeURIComponent(amount)}&description=${encodeURIComponent(description)}`,
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка конвертации');
            return response.json();
        })
        .then(data => {
            loadAccounts();
            alert('Конвертация выполнена успешно');
            convertForm.reset();
            document.getElementById('convert-amount-to').value = '';
            document.getElementById('exchange-rate-display').textContent = 'Курс: -';
        })
        .catch(error => {
            alert(error.message);
        });
    });

    // обработчики изменения выбранных счетов
    document.getElementById('convert-from').addEventListener('change', function() {
        updateCurrencySymbols();
        updateExchangeRate();
        toggleAmountFields();
        clearAmountFields();
    });

    document.getElementById('convert-to').addEventListener('change', function() {
        updateCurrencySymbols();
        updateExchangeRate();
        toggleAmountFields();
        clearAmountFields();
    });

    // обработчики изменения сумм
    document.getElementById('convert-amount').addEventListener('input', function() {
        if (currentExchangeRate > 0) {
            const amount = parseFloat(this.value) || 0;
            document.getElementById('convert-amount-to').value = (amount * currentExchangeRate).toFixed(2);
        }
    });

    document.getElementById('convert-amount-to').addEventListener('input', function() {
        if (currentExchangeRate > 0) {
            const amount = parseFloat(this.value) || 0;
            document.getElementById('convert-amount').value = (amount / currentExchangeRate).toFixed(2);
        }
    });

    function updateCurrencySymbols() {
        const fromSelect = document.getElementById('convert-from');
        const toSelect = document.getElementById('convert-to');
        
        if (fromSelect.selectedIndex > 0) {
            const selectedOption = fromSelect.options[fromSelect.selectedIndex];
            currentFromCurrency = selectedOption.textContent.match(/\(([A-Z]{3})\)/)[1];
            document.getElementById('from-currency-symbol').textContent = getCurrencySymbol(currentFromCurrency);
        } else {
            currentFromCurrency = '';
            document.getElementById('from-currency-symbol').textContent = '';
        }
        
        if (toSelect.selectedIndex > 0) {
            const selectedOption = toSelect.options[toSelect.selectedIndex];
            currentToCurrency = selectedOption.textContent.match(/\(([A-Z]{3})\)/)[1];
            document.getElementById('to-currency-symbol').textContent = getCurrencySymbol(currentToCurrency);
        } else {
            currentToCurrency = '';
            document.getElementById('to-currency-symbol').textContent = '';
        }
    }

    function getCurrencySymbol(currencyCode) {
        const symbols = {
            'USD': '$',
            'EUR': '€',
            'GBP': '£',
            'RUB': '₽',
            'JPY': '¥',
            'CNY': '¥'
        };
        return symbols[currencyCode] || currencyCode;
    }

    function toggleAmountFields() {
        const fromSelected = document.getElementById('convert-from').selectedIndex > 0;
        const toSelected = document.getElementById('convert-to').selectedIndex > 0;
        const bothSelected = fromSelected && toSelected;
        
        document.getElementById('convert-amount').disabled = !bothSelected;
        document.getElementById('convert-amount-to').disabled = !bothSelected;
        
        if (!bothSelected) {
            document.getElementById('convert-amount').value = '';
            document.getElementById('convert-amount-to').value = '';
        }
    }

    function clearAmountFields() {
        document.getElementById('convert-amount').value = '';
        document.getElementById('convert-amount-to').value = '';
    }

    function updateExchangeRate() {
        clearAmountFields();
        if (currentFromCurrency && currentToCurrency && currentFromCurrency !== currentToCurrency) {
            fetch(`/payments/currency-rate/get?from=${currentFromCurrency}&to=${currentToCurrency}`, {
                method: 'GET',
                credentials: 'include'
            })
            .then(response => {
                if (!response.ok) throw new Error('Ошибка получения курса');
                return response.json();
            })
            .then(data => {
                currentExchangeRate = data.rate; // Изменено с data.Rate на data.rate
                document.getElementById('exchange-rate-display').textContent = 
                    `Курс: 1 ${currentFromCurrency} = ${currentExchangeRate.toFixed(4)} ${currentToCurrency}`;
            })
            .catch(error => {
                console.error('Error getting exchange rate:', error);
                document.getElementById('exchange-rate-display').textContent = 'Курс: не доступен';
                currentExchangeRate = 0;
            });
        } else if (currentFromCurrency && currentToCurrency && currentFromCurrency === currentToCurrency) {
            currentExchangeRate = 1;
            document.getElementById('exchange-rate-display').textContent = 'Курс: 1:1 (одинаковые валюты)';
        } else {
            document.getElementById('exchange-rate-display').textContent = 'Курс: -';
            currentExchangeRate = 0;
        }
    }
    
    // Функция загрузки счетов для выпадающих списков
      function loadAccounts() {
        fetch('/account/get', {
            method: 'GET',
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка загрузки счетов');
            return response.json();
        })
        .then(accounts => {
            const transferFromSelect = document.getElementById('transfer-from');
            const depositAccountSelect = document.getElementById('deposit-account');
            const convertFromSelect = document.getElementById('convert-from');
            const convertToSelect = document.getElementById('convert-to');
            
            // Очищаем существующие опции (кроме первой)
            [transferFromSelect, depositAccountSelect, convertFromSelect, convertToSelect].forEach(select => {
                while (select.options.length > 1) {
                    select.remove(1);
                }
            });
            
            // Добавляем счета в выпадающие списки
            accounts.forEach(account => {
                const option = document.createElement('option');
                option.value = account.id;  // Используем account.id вместо account.account_id
                option.textContent = `${account.account_name} (${account.currency}) - ${account.balance.toFixed(2)}`;
                
                transferFromSelect.appendChild(option.cloneNode(true));
                depositAccountSelect.appendChild(option.cloneNode(true));
                convertFromSelect.appendChild(option.cloneNode(true));
                convertToSelect.appendChild(option.cloneNode(true));
            });
        })
        .catch(error => {
            console.error('Error loading accounts:', error);
            alert('Не удалось загрузить счета: ' + error.message);
        });
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