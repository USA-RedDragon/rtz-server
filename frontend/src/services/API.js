import axios from 'axios';

const baseURL = '/api/v1';

const instance = axios.create({
  baseURL,
  withCredentials: true,
});

instance.interceptors.response.use(
  (response) => {
    return response;
  },
  (error) => {
    if (error.response === undefined) {
      return Promise.reject(error);
    }
    const status = error.response.status;
    if (
      window.location.pathname.startsWith('/admin') &&
      (status === 401 || status === 403)
    ) {
      window.location = '/login';
    }

    return Promise.reject(error);
  },
);

export default instance;
