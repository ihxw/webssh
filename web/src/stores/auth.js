import { defineStore } from 'pinia'
import { login as apiLogin, logout as apiLogout, getCurrentUser } from '../api/auth'

export const useAuthStore = defineStore('auth', {
    state: () => ({
        user: null,
        token: localStorage.getItem('token') || null,
    }),

    getters: {
        isAuthenticated: (state) => !!state.token,
        isAdmin: (state) => state.user?.role === 'admin',
    },

    actions: {
        async login(username, password, remember = false) {
            try {
                const response = await apiLogin(username, password, remember)
                this.token = response.token
                this.user = response.user

                // Store token in localStorage
                localStorage.setItem('token', response.token)

                return response
            } catch (error) {
                throw error
            }
        },

        async logout() {
            try {
                await apiLogout()
            } catch (error) {
                console.error('Logout error:', error)
            } finally {
                this.token = null
                this.user = null
                localStorage.removeItem('token')
            }
        },

        async fetchCurrentUser() {
            try {
                const user = await getCurrentUser()
                this.user = user
                return user
            } catch (error) {
                // Token invalid, clear auth state
                this.token = null
                this.user = null
                localStorage.removeItem('token')
                throw error
            }
        },

        setToken(token) {
            this.token = token
            localStorage.setItem('token', token)
        },

        clearAuth() {
            this.token = null
            this.user = null
            localStorage.removeItem('token')
        }
    }
})
