class TokenRefreshManager {
    constructor() {
        this.refreshInterval = 11 * 60 * 1000; 
        this.retryDelay = 10000; 
        this.timerId = null;
    }

    async refreshToken() {
        try {
            const response = await fetch('/auth/refresh', {
                method: 'POST',
                credentials: 'include',
                headers: { 'Content-Type': 'application/json' }
            });

            if (!response.ok) throw new Error(`HTTP ${response.status}`);
            
            console.log('Token refreshed at', new Date().toLocaleTimeString());
            this.startTimer();

        } catch (error) {
            console.error('Refresh failed, retrying...', error);
            this.timerId = setTimeout(() => this.refreshToken(), this.retryDelay);
        }
    }

    startTimer() {
        this.stopTimer();
        this.timerId = setTimeout(() => this.refreshToken(), this.refreshInterval);
    }

    stopTimer() {
        if (this.timerId) clearTimeout(this.timerId);
    }

    start() {
        this.stopTimer();
        this.refreshToken();
    }

    stop() {
        this.stopTimer();
    }
}

// Проверяем и инициализируем глобально
document.addEventListener('DOMContentLoaded', () => {
    if (!window._tokenManager) {
        window._tokenManager = new TokenRefreshManager();
    }
    window._tokenManager.start(); 
});