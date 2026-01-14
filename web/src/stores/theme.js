import { defineStore } from 'pinia'
import { theme } from 'ant-design-vue'

export const useThemeStore = defineStore('theme', {
    state: () => ({
        isDark: false, // Default to light theme
    }),

    getters: {
        themeAlgorithm: (state) => state.isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
        themeToken: (state) => ({
            colorPrimary: '#1890ff',
            colorBgContainer: state.isDark ? '#1f1f1f' : '#ffffff',
            colorBgElevated: state.isDark ? '#1f1f1f' : '#ffffff',
            colorBorder: state.isDark ? '#303030' : '#d9d9d9',
        })
    },

    actions: {
        toggleTheme() {
            this.isDark = !this.isDark
            localStorage.setItem('theme', this.isDark ? 'dark' : 'light')
        },

        initTheme() {
            const savedTheme = localStorage.getItem('theme')
            if (savedTheme) {
                this.isDark = savedTheme === 'dark'
            }
        }
    }
})
