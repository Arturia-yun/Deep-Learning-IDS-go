import axios from 'axios'

const instance = axios.create({
    baseURL: 'http://localhost:8080/api',
    timeout: 5000,
})

export default instance

export const getHistory = (limit = 50) => {
    return instance.get(`/alerts?limit=${limit}`)
}

export const getStatus = () => {
    return instance.get('/status')
}
