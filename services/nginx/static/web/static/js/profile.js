document.addEventListener('DOMContentLoaded', function() {
    const editNameBtn = document.querySelector('.edit-btn[data-field="name"]');
    const editEmailBtn = document.querySelector('.edit-btn[data-field="email"]');
    const editPasswordBtn = document.querySelector('.edit-btn[data-field="password"]');
    
    const editNameModal = document.getElementById('edit-name-modal');
    const editEmailModal = document.getElementById('edit-email-modal');
    const editPasswordModal = document.getElementById('edit-password-modal');
    
    const closeButtons = document.querySelectorAll('.modal .close');
    
    const editNameForm = document.getElementById('edit-name-form');
    const editEmailForm = document.getElementById('edit-email-form');
    const editPasswordForm = document.getElementById('edit-password-form');
    
    // Загрузка данных профиля при открытии страницы
    loadProfile();
    
    // Открытие модальных окон для редактирования
    editNameBtn.addEventListener('click', function() {
        editNameModal.classList.remove('hidden');
    });
    
    editEmailBtn.addEventListener('click', function() {
        editEmailModal.classList.remove('hidden');
    });
    
    editPasswordBtn.addEventListener('click', function() {
        editPasswordModal.classList.remove('hidden');
    });
    
    // Закрытие модальных окон
    closeButtons.forEach(btn => {
        btn.addEventListener('click', function() {
            this.closest('.modal').classList.add('hidden');
        });
    });
    
    // Обработка изменения имени
    editNameForm.addEventListener('submit', function(e) {
        e.preventDefault();
        
        const newName = document.getElementById('new-name').value;
        
        fetch('/user/profile/changename', {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: `name=${encodeURIComponent(newName)}`,
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка изменения имени');
            return response.json();
        })
        .then(data => {
            document.getElementById('profile-name').textContent = newName;
            editNameModal.classList.add('hidden');
            editNameForm.reset();
            alert('Имя успешно изменено');
        })
        .catch(error => {
            alert(error.message);
        });
    });
    
    // Обработка изменения email
    editEmailForm.addEventListener('submit', function(e) {
        e.preventDefault();
        
        const newEmail = document.getElementById('new-email').value;
        
        fetch('/user/profile/changeemail', {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: `email=${encodeURIComponent(newEmail)}`,
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка изменения email');
            return response.json();
        })
        .then(data => {
            document.getElementById('profile-email').textContent = newEmail;
            editEmailModal.classList.add('hidden');
            editEmailForm.reset();
            alert('Email успешно изменён');
        })
        .catch(error => {
            alert(error.message);
        });
    });
    
    // Обработка изменения пароля
    editPasswordForm.addEventListener('submit', function(e) {
        e.preventDefault();
        
        const currentPassword = document.getElementById('current-password').value;
        const newPassword = document.getElementById('new-password').value;
        const confirmPassword = document.getElementById('confirm-password').value;
        
        if (newPassword !== confirmPassword) {
            alert('Новый пароль и подтверждение не совпадают');
            return;
        }
        
        fetch('/user/profile/changepassword', {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: `password=${encodeURIComponent(newPassword)}`,
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка изменения пароля');
            return response.json();
        })
        .then(data => {
            editPasswordModal.classList.add('hidden');
            editPasswordForm.reset();
            alert('Пароль успешно изменён');
        })
        .catch(error => {
            alert(error.message);
        });
    });
    
    // Функция загрузки данных профиля
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
            document.getElementById('profile-name').textContent = profile.full_name;
            document.getElementById('profile-email').textContent = profile.email;
            document.getElementById('username').textContent = profile.full_name;
            
            // Заполняем поля в формах редактирования
            document.getElementById('new-name').value = profile.full_name;
            document.getElementById('new-email').value = profile.email;
        })
        .catch(error => {
            console.error('Error loading profile:', error);
        });
    }
});