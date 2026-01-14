import api from './index'

export const login = async (username, password, remember = false) => {
    return await api.post('/auth/login', { username, password, remember })
}

export const logout = async () => {
    return await api.post('/auth/logout')
}

export const getCurrentUser = async () => {
    return await api.get('/auth/me')
}

export const getWSTicket = async () => {
    return await api.post('/auth/ws-ticket')
}
