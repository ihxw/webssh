import axios from 'axios'
import { message } from 'ant-design-vue'

// Create axios instance
const api = axios.create({
    baseURL: '/api',
    timeout: 30000,
})

// Request interceptor
api.interceptors.request.use(
    (config) => {
        // Add token to headers
        const token = localStorage.getItem('token')
        if (token) {
            config.headers.Authorization = `Bearer ${token}`
        }
        return config
    },
    (error) => {
        return Promise.reject(error)
    }
)

// Response interceptor
api.interceptors.response.use(
    (response) => {
        // Return data directly if success
        if (response.data.success) {
            return response.data.data
        }
        return response.data
    },
    (error) => {
        // Extract error message
        let errorMessage = 'Request failed'
        if (error.response && error.response.data && error.response.data.error) {
            errorMessage = error.response.data.error
        } else if (error.message) {
            errorMessage = error.message
        }

        // Handle errors
        if (error.response) {
            const { status, config } = error.response

            if (status === 401) {
                // If we are already on the login page or trying to login, just show the error
                const isLoginRequest = config.url.includes('/auth/login')
                const isLoginPage = window.location.pathname === '/login' || window.location.hash === '#/login'

                if (isLoginRequest) {
                    message.error(errorMessage || 'Invalid username or password')
                } else {
                    localStorage.removeItem('token')
                    if (!isLoginPage) {
                        message.error('Session expired, please login again')
                        // We use href to force a clean state redirect for session expiration
                        window.location.href = '/login'
                    }
                }
            } else {
                message.error(errorMessage)
            }
        } else if (error.request) {
            message.error(errorMessage || 'Network error, please check your connection')
        } else {
            message.error(errorMessage)
        }

        return Promise.reject(error)
    }
)

export default api
