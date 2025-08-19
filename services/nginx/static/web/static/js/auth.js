document.addEventListener('DOMContentLoaded', function() {
    const loginForm = document.getElementById('login-form');
    const registerForm = document.getElementById('register-form');
    const showRegister = document.getElementById('show-register');
    const showLogin = document.getElementById('show-login');
    const loginBtn = document.getElementById('login-btn');
    const registerBtn = document.getElementById('register-btn');

    // Переключение между формами входа и регистрации
    showRegister.addEventListener('click', function(e) {
        e.preventDefault();
        loginForm.classList.add('hidden');
        registerForm.classList.remove('hidden');
    });

    showLogin.addEventListener('click', function(e) {
        e.preventDefault();
        registerForm.classList.add('hidden');
        loginForm.classList.remove('hidden');
    });

    checkAuthAndRedirect();

    // Обработка входа
    loginBtn.addEventListener('click', function() {
        const email = document.getElementById('login-email').value;
        const password = document.getElementById('login-password').value;

        fetch('/auth/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: `email=${encodeURIComponent(email)}&password=${encodeURIComponent(password)}`
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка входа');
            return response.json();
        })
        .then(data => {
            // Проверяем роль пользователя и перенаправляем
            checkUserRoleAndRedirect();
        })
        .catch(error => {
            alert(error.message);
        });
    });

    // Обработка регистрации
    registerBtn.addEventListener('click', function() {
        const name = document.getElementById('register-name').value;
        const email = document.getElementById('register-email').value;
        const password = document.getElementById('register-password').value;

        fetch('/auth/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: `name=${encodeURIComponent(name)}&email=${encodeURIComponent(email)}&password=${encodeURIComponent(password)}`
        })
        .then(async response => {
            if (!response.ok) {
                const errorData = await response.json();
                // Парсим строку ошибки для поиска конкретных сообщений
                const errorText = typeof errorData === 'string' ? errorData : JSON.stringify(errorData);
                if (errorText.includes('invalid email format')) {
                    throw new Error('Неверный формат email адреса');
                } else if (errorText.includes('password must be at least 8 characters')) {
                    throw new Error('Пароль должен содержать минимум 8 символов, включая заглавные и строчные буквы, цифры и специальные символы');
                } else {
                    throw new Error('Ошибка регистрации: ' + errorText);
                }
            }
            return response.json();
        })
        .then(data => {
            window.location.href = '/accounts';
        })
        .catch(error => {
            alert(error.message);
        });
    });
    
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
                window.location.href = '/support';
            } else {
                window.location.href = '/accounts';
            }
        })
        .catch(error => {
            console.error('Error checking role:', error);
            // По умолчанию перенаправляем на страницу пользователя
            window.location.href = '/accounts';
        });
    }

    function checkAuthAndRedirect() {
        // Попробуем запросить информацию, доступную только авторизованным пользователям
        fetch('/user/profile', {
            method: 'GET',
            credentials: 'include'
        })
        .then(response => {
            if (response.ok) {
                // Если запрос прошел успешно, пользователь авторизован
                checkUserRoleAndRedirect();
            }
        });
    }
})