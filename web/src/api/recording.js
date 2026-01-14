import api from './index'

export const listRecordings = async () => {
    return await api.get('/recordings')
}

export const deleteRecording = async (id) => {
    return await api.delete(`/recordings/${id}`)
}

export const getRecordingStreamUrl = (id) => {
    const token = localStorage.getItem('token')
    return `/api/recordings/${id}/stream?token=${token}`
}
