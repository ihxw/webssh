import api from './index'

export const listFiles = async (hostId, path = '.') => {
    return await api.get(`/sftp/list/${hostId}`, { params: { path } })
}

export const downloadFile = (hostId, path) => {
    // For download, we use a direct window.open or a hidden link because it's a stream
    const token = localStorage.getItem('token')
    const url = `/api/sftp/download/${hostId}?path=${encodeURIComponent(path)}&token=${token}`
    window.open(url, '_blank')
}

export const uploadFile = async (hostId, path, file) => {
    const formData = new FormData()
    formData.append('path', path)
    formData.append('file', file)
    return await api.post(`/sftp/upload/${hostId}`, formData, {
        headers: {
            'Content-Type': 'multipart/form-data'
        }
    })
}

export const deleteFile = async (hostId, path) => {
    return await api.delete(`/sftp/delete/${hostId}`, { params: { path } })
}
