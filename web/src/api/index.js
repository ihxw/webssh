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
        // Handle errors
        if (error.response) {
            const { status, data } = error.response

            if (status === 401) {
                // Unauthorized, clear token and redirect to login
                localStorage.removeItem('token')
                window.location.href = '/login'
                message.error('Session expired, please login again')
            } else if (status === 403) {
                message.error(data.error || 'Access denied')
            } else if (status === 404) {
                message.error(data.error || 'Resource not found')
            } else if (status >= 500) {
                message.error(data.error || 'Server error')
            } else {
                message.error(data.error || 'Request failed')
            }
        } else if (error.request) {
            message.error('Network error, please check your connection')
        } else {
            message.error('Request failed: ' + error.message)
        }

        return Promise.reject(error)
    }
)

export default api
