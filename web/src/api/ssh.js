import api from './index'

export const getHosts = async (filters = {}) => {
    const params = new URLSearchParams(filters).toString()
    return await api.get(`/ssh-hosts${params ? '?' + params : ''}`)
}

export const getHost = async (id) => {
    return await api.get(`/ssh-hosts/${id}`)
}

export const createHost = async (hostData) => {
    return await api.post('/ssh-hosts', hostData)
}

export const updateHost = async (id, hostData) => {
    return await api.put(`/ssh-hosts/${id}`, hostData)
}

export const deleteHost = async (id) => {
    return await api.delete(`/ssh-hosts/${id}`)
}
