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
        .then(response => {
            if (!response.ok) throw new Error('Ошибка регистрации');
            return response.json();
        })
        .then(data => {
                // Перенаправляем на страницу пользователя
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