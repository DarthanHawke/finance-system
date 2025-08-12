document.addEventListener('DOMContentLoaded', function() {
    // Обработка выхода из системы
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
                window.location.href = '/support';
            } else {
                //пропускаем
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